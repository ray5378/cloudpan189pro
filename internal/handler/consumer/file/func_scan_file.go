package file

import (
	"fmt"
	"strings"

	"github.com/xxcheng123/cloudpan189-share/internal/services/filetasklog"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"

	cloudbridgeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/taskcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/datatypes"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/types/apierrcode"
	"github.com/xxcheng123/cloudpan189-share/internal/types/converter"
	"github.com/xxcheng123/cloudpan189-share/internal/types/topic"
	"go.uber.org/zap"
)

func (h *handler) ScanFile() taskcontext.HandlerFunc {
	return func(ctx *taskcontext.Context) (scanErr error) {
		req := new(topic.FileScanFileRequest)

		if err := ctx.Unmarshal(req); err != nil {
			return err
		}

		var (
			logger = ctx.GetContext().Logger
		)

		topFile := models.RootFile()

		if req.FileId != 0 {
			if file, err := h.virtualFileService.Query(ctx.GetContext(), req.FileId); err != nil {
				logger.Error("查询文件失败", zap.Int64("file_id", req.FileId), zap.Error(err))

				return err
			} else {
				topFile = file
			}
		}

		if !topFile.IsDir {
			ctx.GetContext().Error("文件不是文件夹", zap.Int64("file_id", req.FileId))

			return errors.New("文件不是文件夹，不支持扫描")
		}

		tracker, logErr := h.fileTaskLogService.Create(
			ctx.GetContext(),
			req.Topic().String(),
			fmt.Sprintf("扫描目录: %s", ctx.GetContext().String(consts.CtxKeyFullPath, topFile.Name)),
			filetasklog.WithFile(topFile.ID),
			filetasklog.WithDesc(fmt.Sprintf(
				"调用者: %s, 深度扫描: %t, 文件ID: %d, 目录名: %s, 上级ID: %d, 挂载点ID: %d",
				ctx.GetContext().String(consts.CtxKeyInvokeHandlerName, "unknown"),
				req.Deep,
				req.FileId,
				topFile.Name,
				topFile.ParentId,
				topFile.TopId,
			)),
		)
		if logErr != nil {
			logger.Error("创建文件任务日志失败", zap.Int64("file_id", req.FileId), zap.Error(logErr))

			return logErr
		}

		_ = h.fileTaskLogService.Running(ctx.GetContext(), tracker)

		defer func() {
			if scanErr != nil {
				_ = h.fileTaskLogService.Failed(ctx.GetContext(), tracker, tracker.WithCost(), utils.WithField("result", scanErr.Error()))
			} else if err := h.fileTaskLogService.Completed(ctx.GetContext(), tracker, tracker.WithCost()); err != nil {
				logger.Error("更新文件任务日志失败", zap.Int64("file_id", req.FileId), zap.Error(err))
			}
		}()

		if shared.MediaConfig != nil && shared.MediaConfig.Enable && shared.MediaConfig.AutoClean {
			defer func() {
				_ = h.mediaFileService.ClearEmptyDir(ctx.GetContext(), shared.MediaConfig.StoragePath)
			}()
		}

		logger.Debug("开始扫描文件", zap.Int64("file_id", req.FileId), zap.Bool("deep", req.Deep))

		if err := h.walkFile(ctx.GetContext(), req.FileId, func(ctx context.Context, inputFile *models.VirtualFile, childrenFiles []*models.VirtualFile) (nextWalkFiles []*models.VirtualFile, err error) {
			if inputFile == nil {
				return make([]*models.VirtualFile, 0), nil
			}

			_ = h.fileTaskLogService.FlushCount(ctx, tracker, filetasklog.WithTotalCounter(1))
			defer func() {
				_ = h.fileTaskLogService.FlushCount(ctx, tracker, filetasklog.WithCompletedOneCounter())
			}()

			// 如果是根目录，直接返回子目录
			if inputFile.ID == 0 || inputFile.OsType == models.OsTypeFolder {
				ctx.Info("根目录或者目录直接返回子目录查询")

				return lo.Filter(childrenFiles, func(item *models.VirtualFile, index int) bool {
					return item.IsDir
				}), nil
			}

			var fileConverters []converter.VirtualFileConverter

			switch inputFile.OsType {
			case models.OsTypeSubscribe:
				var (
					upUserId string
					ok       bool
				)

				if upUserId, ok = inputFile.Addition.String(consts.FileAdditionKeyUpUserId); !ok {
					ctx.Error("获取订阅用户失败", zap.Int64("file_id", inputFile.ID))

					return nil, errors.New("获取订阅用户失败")
				}

				fileConverters, err = h.cloudBridgeService.GetSubscribeUserFiles(ctx, upUserId)
			case models.OsTypeSubscribeShareFolder:
				var (
					upUserId string
					shareId  int64
					isFolder bool
					ok       bool
				)

				if upUserId, ok = inputFile.Addition.String(consts.FileAdditionKeyUpUserId); !ok {
					ctx.Error("获取订阅用户失败", zap.Int64("file_id", inputFile.ID))

					return nil, errors.New("获取订阅用户失败")
				}

				if shareId, ok = inputFile.Addition.Int64(consts.FileAdditionKeyShareId); !ok {
					ctx.Error("获取分享ID失败", zap.Int64("file_id", inputFile.ID))

					return nil, errors.New("获取分享ID失败")
				}

				if isFolder, ok = inputFile.Addition.Bool(consts.FileAdditionKeyIsFolder); !ok {
					ctx.Error("获取分享类型失败", zap.Int64("file_id", inputFile.ID))

					return nil, errors.New("获取分享类型失败")
				}

				mountInfo, mountErr := h.mountPointService.Query(ctx, inputFile.TopId)
				if mountErr != nil {
					ctx.Error("查询挂载点失败", zap.Int64("file_id", inputFile.ID), zap.Error(mountErr))

					return nil, mountErr
				}

				token, queryErr := h.cloudTokenService.Query(ctx, mountInfo.TokenId)
				if queryErr != nil {
					ctx.Error("获取云盘令牌失败", zap.Int64("file_id", inputFile.ID), zap.Error(queryErr))

					return nil, errors.New("获取云盘令牌失败")
				}

				fileConverters, err = h.cloudBridgeService.GetSubscribeShareFiles(ctx, cloudbridgeSvi.NewAuthToken(token.AccessToken, token.ExpiresIn), upUserId, shareId, inputFile.CloudId, isFolder)
			case models.OsTypeShareFolder:
				var (
					shareId    int64
					shareMode  int
					accessCode string
					isFolder   bool
					ok         bool
				)

				if shareId, ok = inputFile.Addition.Int64(consts.FileAdditionKeyShareId); !ok {
					ctx.Error("获取分享ID失败", zap.Int64("file_id", inputFile.ID))
					return nil, errors.New("获取分享ID失败")
				}

				if shareMode, ok = inputFile.Addition.Int(consts.FileAdditionKeyShareMode); !ok {
					if v, fOk := inputFile.Addition[consts.FileAdditionKeyShareMode]; fOk {
						if fMode, ok := v.(float64); ok {
							shareMode = int(fMode)
						} else {
							shareMode = 1
						}
					} else {
						shareMode = 1
					}
				}

				accessCode, _ = inputFile.Addition.String(consts.FileAdditionKeyAccessCode)
				// 调试
				logger.Info("准备扫描分享文件",
					zap.Int64("shareId", shareId),
					zap.String("accessCode", accessCode),
					zap.Int("shareMode", shareMode))

				if isFolder, ok = inputFile.Addition.Bool(consts.FileAdditionKeyIsFolder); !ok {
					ctx.Error("获取分享类型失败", zap.Int64("file_id", inputFile.ID))
					return nil, errors.New("获取分享类型失败")
				}

				fileConverters, err = h.cloudBridgeService.GetShareFiles(ctx, shareId, inputFile.CloudId, shareMode, accessCode, isFolder)
			case models.OsTypePersonFolder:
				mountInfo, mountErr := h.mountPointService.Query(ctx, inputFile.TopId)
				if mountErr != nil {
					ctx.Error("查询挂载点失败", zap.Int64("file_id", inputFile.ID), zap.Error(mountErr))

					return nil, mountErr
				}

				token, queryErr := h.cloudTokenService.Query(ctx, mountInfo.TokenId)
				if queryErr != nil {
					ctx.Error("获取云盘令牌失败", zap.Int64("file_id", inputFile.ID), zap.Error(err))

					return nil, errors.New("获取云盘令牌失败")
				}

				fileConverters, err = h.cloudBridgeService.GetCloudFiles(ctx, cloudbridgeSvi.NewAuthToken(token.AccessToken, token.ExpiresIn), inputFile.CloudId)
			case models.OsTypeFamilyFolder:
				mountInfo, mountErr := h.mountPointService.Query(ctx, inputFile.TopId)
				if mountErr != nil {
					ctx.Error("查询挂载点失败", zap.Int64("file_id", inputFile.ID), zap.Error(mountErr))

					return nil, mountErr
				}

				token, queryErr := h.cloudTokenService.Query(ctx, mountInfo.TokenId)
				if queryErr != nil {
					ctx.Error("获取云盘令牌失败", zap.Int64("file_id", inputFile.ID), zap.Error(queryErr))

					return nil, errors.New("获取云盘令牌失败")
				}

				var (
					familyId string
					ok       bool
				)

				if familyId, ok = inputFile.Addition.String(consts.FileAdditionKeyFamilyId); !ok {
					ctx.Error("获取家庭文件夹ID失败", zap.Int64("file_id", inputFile.ID))

					return nil, errors.New("获取家庭文件夹ID失败")
				}

				fileConverters, err = h.cloudBridgeService.GetCloudFamilyFiles(ctx, cloudbridgeSvi.NewAuthToken(token.AccessToken, token.ExpiresIn), familyId, inputFile.CloudId)
			default:
				return nil, errors.Errorf("不支持的文件类型 %s", inputFile.OsType)
			}

			if err != nil {
				ctx.Error("获取新文件失败", zap.Int64("file_id", inputFile.ID), zap.String("os_type", inputFile.OsType), zap.Error(err))

				return nil, errors.Wrap(err, "获取新文件失败")
			}

			var newFiles = make([]*models.VirtualFile, 0, len(fileConverters))
			for _, c := range fileConverters {
				vf := c.TransformVirtualFile(inputFile.TopId, inputFile.ID)
				newFiles = append(newFiles, vf)
			}

			// 递归扫描语义：每一层拿到远端结果后，先遍历整批结果查找 .cas，
			// 不依赖“新增/已存在”差异分支，避免多级目录或已存在文件命不中自动归集。
			for _, scannedFile := range newFiles {
				if scannedFile == nil || scannedFile.IsDir {
					continue
				}
				if !strings.HasSuffix(strings.ToLower(scannedFile.Name), ".cas") {
					continue
				}
				// 优先把当前扫描上下文里的真实父目录路径带给 CAS 自动归集，避免依赖库里偶发错误的父链。
				// 这里必须使用 CtxKeyFileFullPath（当前递归节点的真实路径），不能用入口级别的 CtxKeyFullPath。
				// 注意：这里扫描到的 scannedFile 是“当前节点 inputFile 的子文件”，
				// 所以应直接使用当前节点路径本身；如果这里再 path.Dir 一次，就会把 Season 2 削成上一级目录。
				sourceDirPath := ""
				if currentFilePath, ok := ctx.GetString(consts.CtxKeyFileFullPath); ok && strings.TrimSpace(currentFilePath) != "" {
					sourceDirPath = strings.TrimSpace(currentFilePath)
				}
				if sourceDirPath != "" {
					if scannedFile.Addition == nil {
						scannedFile.Addition = datatypes.JSONMap{}
					}
					scannedFile.Addition[consts.FileAdditionKeySourceDirPath] = sourceDirPath
				}
				ctx.Info("扫描到CAS文件，准备尝试自动归集",
					zap.String("file_name", scannedFile.Name),
					zap.String("cloud_id", scannedFile.CloudId),
					zap.String("os_type", scannedFile.OsType),
					zap.String("source_dir_path", sourceDirPath),
				)
				if err := h.tryCollectCASFromVirtualFile(ctx, scannedFile); err != nil {
					ctx.Error("存储刷新链CAS自动归集失败",
						zap.String("file_name", scannedFile.Name),
						zap.String("cloud_id", scannedFile.CloudId),
						zap.Error(err),
					)
				}
			}

			// 创建映射表，用于快速查找
			var (
				newFileMap = make(map[string]*models.VirtualFile)
				oldFileMap = make(map[string]*models.VirtualFile)
			)

			for _, item := range newFiles {
				key := item.CloudId
				newFileMap[key] = item
			}

			for _, item := range childrenFiles {
				key := item.CloudId
				oldFileMap[key] = item
			}

			var (
				filesToUpdateMap = map[int64][]utils.Field{}
				pid              = inputFile.ID
				filesToCreate    = make([]*models.VirtualFile, 0)
				filesToDelete    = make([]*models.VirtualFile, 0)
				filesToDeep      = make([]*models.VirtualFile, 0)
			)

			// 遍历扫描到的文件，找出新增和更新的文件
			for cloudId, newFile := range newFileMap {
				if oldFile, exists := oldFileMap[cloudId]; exists {
					// 订阅扫描链补字段：即使 rev 不变，也要把新 addition 中补充出来的字段（如 access_url）同步回库。
					mergeAddition := func(oldAddition, newAddition datatypes.JSONMap) (datatypes.JSONMap, bool) {
						merged := make(datatypes.JSONMap, len(oldAddition)+len(newAddition))
						changed := false
						for k, v := range oldAddition {
							merged[k] = v
						}
						for k, v := range newAddition {
							if v == nil {
								continue
							}
							if ov, ok := merged[k]; !ok || ov != v {
								merged[k] = v
								changed = true
							}
						}
						return merged, changed
					}

					if mergedAddition, changed := mergeAddition(oldFile.Addition, newFile.Addition); changed {
						ctx.Debug("文件附加信息更新",
							zap.Int64("file_id", oldFile.ID),
							zap.String("file_name", oldFile.Name),
						)
						filesToUpdateMap[oldFile.ID] = append(filesToUpdateMap[oldFile.ID],
							utils.WithField("addition", mergedAddition),
						)
						oldFile.Addition = mergedAddition
					}

					shouldWalkForCAS := shared.SettingAddition.CasTargetEnabled && shared.SettingAddition.CasAutoCollectEnabled

					// 文件存在，检查是否需要更新（通过Rev比较）
					if oldFile.Rev != newFile.Rev {
						ctx.Debug("文件存在差异 - rev changed",
							zap.Int64("parent_id", pid),
							zap.String("cloud_id", cloudId),
							zap.String("file_name", newFile.Name),
							zap.String("old_rev", oldFile.Rev),
							zap.String("new_rev", newFile.Rev))

						filesToUpdateMap[oldFile.ID] = append(make([]utils.Field, 0, 5),
							utils.WithField("name", utils.SanitizeFileName(newFile.Name)),
							utils.WithField("rev", newFile.Rev),
							utils.WithField("size", newFile.Size),
							utils.WithField("modify_date", newFile.ModifyDate),
							utils.WithField("hash", strings.ToLower(newFile.Hash)),
						)
					} else if oldFile.IsDir && (req.Deep || shouldWalkForCAS) {
						filesToDeep = append(filesToDeep, oldFile)
					}
				} else {
					ctx.Debug("发现新文件",
						zap.Int64("parent_id", pid),
						zap.String("cloud_id", cloudId),
						zap.String("file_name", newFile.Name),
						zap.String("rev", newFile.Rev))
					// 文件不存在，需要新增
					newFile.ParentId = pid
					filesToCreate = append(filesToCreate, newFile)
				}
			}

			// 遍历数据库中的文件，找出需要删除的文件
			for cloudId, dbFile := range oldFileMap {
				if _, exists := newFileMap[cloudId]; !exists &&
					!dbFile.IsTop {
					ctx.Debug("文件不存在 - 删除",
						zap.Int64("parent_id", pid),
						zap.String("cloud_id", cloudId),
						zap.String("file_name", dbFile.Name),
						zap.Int64("file_id", dbFile.ID),
						zap.String("rev", dbFile.Rev))
					// 扫描结果中不存在该文件，需要删除

					filesToDelete = append(filesToDelete, dbFile)
				}
			}

			var (
				createCount = len(filesToCreate)
				deleteCount = len(filesToDelete)
				updateCount = len(filesToUpdateMap)
			)

			_ = h.fileTaskLogService.FlushCount(ctx, tracker,
				filetasklog.WithTotalCounter(createCount),
				filetasklog.WithTotalCounter(deleteCount),
				filetasklog.WithTotalCounter(updateCount),
			)

			var (
				errs []error
			)

			// 文件执行顺序 先删除后新增
			// 特殊情况：
			// 相同目录，example.mkv 待删除，example.mp4 待新增，此时已开启 strm 自动创建
			// 如果先增后删 会导致 example.mp4 对应的 strm 文件无法创建成功

			// 先删除文件
			if len(filesToDelete) > 0 {
				if err = h.batchDeleteFiles(ctx, filesToDelete); err != nil {
					ctx.Error("批量删除文件失败", zap.Error(err))

					errs = append(errs, err)
				}
			}

			_ = h.fileTaskLogService.FlushCount(ctx, tracker, filetasklog.WithCompletedCounter(deleteCount))

			// 新增文件时会有一个问题 一个目录底下有相同文件名的文件 会导致新增失败
			// 原来的解决办法：在新增文件时，直接忽略。但是会有一个问题，后面的扫描会一直显示这个文件不存在，会尝试创建，然后因为开启了 pid 和 name 的唯一索引，会创建失败
			// 现在的解决办法：如果重复就添加一个随机后缀
			// filesToCreate = lo.UniqBy(filesToCreate, func(item *models.VirtualFile) string {
			// 	return item.Name
			// })
			// 但是还有一个特殊情况，原本没有相同文件，后面同级目录添加一个重复的数据进来了，这个时候也会无限重复添加失败。 解决办法：在新增文件前查询
			// for _, file := range filesToCreate {
			// 	if _, exists := uniqFilesToCreateMap[file.Name]; exists {
			// 		uniqFilesToCreateMap[file.Name]++
			// 		file.Name = fmt.Sprintf("%s(%d)", file.Name, uniqFilesToCreateMap[file.Name])
			// 	} else {
			// 		uniqFilesToCreateMap[file.Name] = 1
			// 	}
			// }

			// 新增文件
			if len(filesToCreate) > 0 {
				if err = h.batchCreateFiles(ctx, pid, filesToCreate); err != nil {
					ctx.Error("批量创建文件失败", zap.Error(err))

					errs = append(errs, err)
				}
			}

			_ = h.fileTaskLogService.FlushCount(ctx, tracker, filetasklog.WithCompletedCounter(createCount))

			// 更新文件
			if len(filesToUpdateMap) > 0 {
				if err = h.batchUpdateFiles(ctx, filesToUpdateMap); err != nil {
					ctx.Error("批量更新文件失败", zap.Error(err))

					errs = append(errs, err)
				}
			}

			_ = h.fileTaskLogService.FlushCount(ctx, tracker, filetasklog.WithCompletedCounter(updateCount))

			if len(errs) > 0 {
				return nil, errors.New("文件处理失败")
			}

			// 只有成功创建的目录文件才需要继续遍历（ID > 0）
			createdDirFiles := lo.Filter(filesToCreate, func(item *models.VirtualFile, _ int) bool {
				return item.IsDir && item.ID > 0
			})

			nextWalkFiles = append(createdDirFiles, filesToDeep...)

			if len(nextWalkFiles) > 0 {
				ctx.Debug("继续执行下次遍历",
					zap.Int("next_walk_files_len", len(nextWalkFiles)),
					zap.Bool("deep", req.Deep),
					zap.Bool("cas_collect_enabled", shared.SettingAddition.CasTargetEnabled && shared.SettingAddition.CasAutoCollectEnabled),
				)
			}

			return nextWalkFiles, nil
		}); err != nil {
			ctx.GetContext().Error("执行时有错误", zap.Error(err))

			if apiErr, ok := apierrcode.As(err); ok {
				return apiErr
			}
		}

		return nil
	}
}
