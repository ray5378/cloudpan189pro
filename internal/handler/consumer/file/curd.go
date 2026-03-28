package file

import (
	"path"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	verifySvi "github.com/xxcheng123/cloudpan189-share/internal/services/verify"
	"github.com/xxcheng123/cloudpan189-share/internal/services/virtualfile"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"go.uber.org/zap"
)

// batchDeleteFiles 递归删除文件
func (h *handler) batchDeleteFiles(ctx context.Context, filesToDelete []*models.VirtualFile) (err error) {
	if len(filesToDelete) == 0 {
		return nil
	}

	// 分批删除逻辑：避免一次性删除过多数据导致事务过大
	const batchSize = 100
	total := len(filesToDelete)

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		batch := filesToDelete[i:end]
		ids := make([]int64, 0, len(batch))
		for _, file := range batch {
			ids = append(ids, file.ID)
		}

		// 执行当前批次的删除
		if _, err = h.virtualFileService.BatchDelete(ctx, ids, h.deleteStrmIterator); err != nil {
			ctx.Error("批量删除文件 - 服务层删除失败", zap.Int64s("file_ids", ids), zap.Error(err))
			return err
		}

		// 性能优化：每批次删除后短暂休眠，释放 DB 锁
		time.Sleep(20 * time.Millisecond)
	}

	for _, file := range filesToDelete {
		if file.IsDir {
			currentPath, _ := ctx.GetString(consts.CtxKeyFileFullPath)
			subCtx := ctx.WithValue(consts.CtxKeyFileFullPath, path.Join(currentPath, file.Name))

			if child, childErr := h.virtualFileService.List(subCtx, &virtualfile.ListRequest{
				ParentId: &file.ID,
			}); childErr != nil {
				if !errors.Is(childErr, gorm.ErrRecordNotFound) {
					ctx.Error("批量删除文件 - 服务层查询子节点失败", zap.Int64("file_id", file.ID), zap.Error(childErr))
				}
				continue
			} else if len(child) > 0 {
				if err = h.batchDeleteFiles(subCtx, child); err != nil {
					ctx.Error("批量删除文件 - 子节点删除失败", zap.Int64("file_id", file.ID), zap.Error(err))
					continue
				}
			}
		}
	}

	return nil
}

// clearMountFiles 清理挂载点下的所有文件
func (h *handler) clearMountFiles(ctx context.Context, topId int64) error {
	ctx.Debug("清理挂载文件 - 开始清理", zap.Int64("top_id", topId))

	for {
		files, err := h.virtualFileService.List(ctx, &virtualfile.ListRequest{
			TopId:       &topId,
			CurrentPage: 1,
			PageSize:    200, // 缩小单次查询量
		})
		if err != nil {
			ctx.Error("清理挂载文件 - 服务层查询失败", zap.Int64("top_id", topId), zap.Error(err))
			return err
		}

		if len(files) == 0 {
			break
		}

		filesToDelete := make([]int64, 0, len(files))
		for _, file := range files {
			if file.ID != topId {
				filesToDelete = append(filesToDelete, file.ID)
			}
		}

		if len(filesToDelete) == 0 {
			break
		}

		if _, err = h.virtualFileService.BatchDelete(ctx, filesToDelete, h.deleteStrmIterator); err != nil {
			ctx.Error("批量删除文件 - 服务层删除失败", zap.Int64s("file_ids", filesToDelete), zap.Error(err))
			return err
		}

		time.Sleep(50 * time.Millisecond)
	}

	return nil
}

func (h *handler) batchCreateFiles(ctx context.Context, pid int64, filesToCreate []*models.VirtualFile) (err error) {
	const batchSize = 50 // 每次事务插入 50 条
	total := len(filesToCreate)

	if total == 0 {
		return nil
	}

	ctx.Debug("批量创建文件 - 开始", zap.Int("total_count", total), zap.Int64("pid", pid))

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		batch := filesToCreate[i:end]

		// 执行小批量创建
		// 注意：BatchCreate 内部的事务范围是这一小批，而不是整个 filesToCreate
		_, err = h.virtualFileService.BatchCreate(ctx, pid, batch, h.createStrmIteratorfunc)
		if err != nil {
			ctx.Error("批量创建文件 - 分片创建失败", zap.Int64("pid", pid), zap.Int("start_index", i), zap.Error(err))
			return err
		}

		// 关键优化：每批次休眠 20-50ms，彻底解决 SQLite 锁表导致的 Web 端超时
		time.Sleep(20 * time.Millisecond)
	}

	return nil
}

func (h *handler) batchUpdateFiles(ctx context.Context, filesToUpdate map[int64][]utils.Field) (err error) {
	if err = h.virtualFileService.BatchUpdate(ctx, filesToUpdate); err != nil {
		ctx.Error("批量更新文件 - 服务层更新失败", zap.Error(err))
		return err
	}

	return nil
}

func (h *handler) createStrmIteratorfunc(ctx context.Context, result *gorm.DB, files []*models.VirtualFile) {
	if result.Error == nil && shared.MediaConfig != nil && shared.MediaConfig.Enable {
		dirPath, ok := ctx.GetString(consts.CtxKeyFileFullPath)
		if !ok {
			ctx.Error("批量创建文件 - 获取文件路径失败")

			return
		}

		ctx.Debug("批量创建文件 - 创建 strm 文件", zap.Int("file_count", len(files)), zap.String("full_path", dirPath))

		for _, file := range files {
			if file.IsDir {
				continue
			}

			// 获取文件后缀
			extName := path.Ext(file.Name)
			if len(shared.MediaConfig.IncludedSuffixes) > 0 && !slices.Contains(shared.MediaConfig.IncludedSuffixes, extName) {
				continue
			}

			// 重新生成文件名: vidoe.mp4 -> video.strm
			filename := strings.TrimSuffix(file.Name, extName) + ".strm"

			// 生成URL
			values, err := h.verifyService.SignV1(ctx, file.ID, verifySvi.WithV1NoExpire())
			if err != nil {
				ctx.Error("批量创建文件 - 遍历 - 获取文件签名失败", zap.Int64("file_id", file.ID), zap.Error(err))

				continue
			}

			if id, err := h.mediaFileService.WriteStrm(ctx, shared.MediaConfig.GetCar(dirPath, filename), file.ID, shared.JoinDownloadURL(file.ID, values)); err != nil {
				ctx.Error("批量创建文件 - 遍历 - 创建 strm 文件失败", zap.Int64("file_id", file.ID), zap.Error(err))

				continue
			} else {
				ctx.Debug("批量创建文件 - 遍历 - 创建 strm 文件成功", zap.Int64("file_id", file.ID), zap.Int64("media_file_id", id))
			}
		}
	}
}

func (h *handler) deleteStrmIterator(ctx context.Context, result *gorm.DB, files []*models.VirtualFile) {
	if result.Error == nil && shared.MediaConfig != nil && shared.MediaConfig.Enable {
		ctx.Debug("批量删除文件 - 删除 strm 文件", zap.Int("file_count", len(files)))

		dirPath, hasPath := ctx.GetString(consts.CtxKeyFileFullPath)

		for _, file := range files {
			if file.IsDir {
				continue
			}

			_ = h.mediaFileService.DeleteStrm(ctx, file.ID, shared.MediaConfig.StoragePath)

			if hasPath {
				extName := path.Ext(file.Name)
				if len(shared.MediaConfig.IncludedSuffixes) > 0 && !slices.Contains(shared.MediaConfig.IncludedSuffixes, extName) {
					continue
				}

				strmName := strings.TrimSuffix(file.Name, extName) + ".strm"
				fullPhysicalPath := path.Join(shared.MediaConfig.StoragePath, dirPath, strmName)

				if err := h.mediaFileService.DeleteStrmByFullPath(ctx, fullPhysicalPath); err != nil {
					ctx.Warn("删除 strm 物理文件失败", zap.String("path", fullPhysicalPath), zap.Error(err))
				}
			}
		}
	}
}
