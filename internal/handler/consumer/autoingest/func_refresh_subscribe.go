package autoingest

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/taskcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/datatypes"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/types/autoingest"
	"github.com/xxcheng123/cloudpan189-share/internal/types/topic"
	"go.uber.org/zap"
	"gorm.io/gorm"

	cloudbridgeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudbridge"
	storagefacadeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/storagefacade"
)

func (h *handler) RefreshSubscribe() taskcontext.HandlerFunc {
	return func(ctx *taskcontext.Context) error {
		req := new(topic.AutoIngestRefreshSubscribeRequest)

		if err := ctx.Unmarshal(req); err != nil {
			return err
		}

		var (
			pageSize   = 30
			pageNum    = 1
			hasTop     = true
			shouldNext = true

			addCount    int64 = 0
			failedCount int64 = 0
		)

		logger := ctx.GetContext().Logger

		// 去查询入库计划
		plan, err := h.autoIngestPlanService.Query(ctx.GetContext(), req.PlanId)
		if err != nil {
			logger.Error("查询入库计划信息失败", zap.Error(err), zap.Int64("plan_id", req.PlanId))

			return err
		}

		var nextOffset = plan.Offset

		if plan.SourceType != autoingest.SourceTypeSubscribe {
			logger.Error("计划类型错误", zap.String("source_type", plan.SourceType.String()))

			return errors.New("计划类型错误")
		}

		addition := new(models.AutoIngestPlanSubscribeAddition)
		_ = plan.Addition.Unmarshal(addition)

		defer func() {
			// 执行完了 更新 数据
			_ = h.autoIngestPlanService.UpdateOffset(ctx.GetContext(), req.PlanId, nextOffset)
			_ = h.autoIngestPlanService.IncrAddCount(ctx.GetContext(), req.PlanId, addCount)
			_ = h.autoIngestPlanService.IncrFailedCount(ctx.GetContext(), req.PlanId, failedCount)
		}()

		for shouldNext {
			list, _, err := h.cloudbridgeService.GetSubscribeUserShareResource(ctx.GetContext(), addition.UpUserId, func(opt *cloudbridgeSvi.SubscribeUserShareResourceOption) {
				opt.PageNum = pageNum
				opt.PageSize = pageSize
			})
			if err != nil {
				logger.Error("获取订阅号内容时失败了~", zap.String("up_user_id", addition.UpUserId), zap.Int("page_num", pageNum), zap.Int("page_size", pageSize))

				return err
			}

			if len(list) == 0 {
				break
			}

			pageNum++

			for _, item := range list {
				if item.IsTop != 1 {
					hasTop = false
				}

				itemOffset := item.ShareTime.Unix()
				if itemOffset > nextOffset {
					nextOffset = itemOffset
				}

				if itemOffset <= plan.Offset {
					if !hasTop {
						shouldNext = false
					}

					continue
				}

				logger.Debug("发现新的待入库文件", zap.String("name", item.Name))

				// 检查这个文件存不存在先
				fullPath := path.Join(plan.ParentPath, item.Name)
				if _, err := h.virtualFileService.QueryByPath(ctx.GetContext(), fullPath); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					logger.Error("查询虚拟文件路径失败 跳过本次自动入库", zap.String("path", fullPath), zap.Error(err))

					continue
				} else if err == nil && plan.OnConflict == autoingest.OnConflictRename {
					fullPath = path.Join(plan.ParentPath, fmt.Sprintf("%s_%d", item.Name, time.Now().Unix()))
				}

				id, err := h.storageFacadeService.CreateStorage(ctx.GetContext(),
					&storagefacadeSvi.CreateStorageRequest{
						LocalPath:  fullPath,
						OsType:     models.OsTypeSubscribeShareFolder,
						CloudToken: plan.TokenId,
						FileId:     item.ID,
						Addition: datatypes.JSONMap{
							consts.FileAdditionKeyUpUserId: addition.UpUserId,
							consts.FileAdditionKeyShareId:  item.ShareId,
							consts.FileAdditionKeyIsFolder: item.IsFolder,
						},
						EnableAutoRefresh: plan.RefreshStrategy.EnableAutoRefresh,
						EnableDeepRefresh: plan.RefreshStrategy.EnableDeepRefresh,
						AutoRefreshDays:   plan.RefreshStrategy.AutoRefreshDays,
						RefreshInterval:   plan.RefreshStrategy.RefreshInterval,
					},
				)
				if err != nil {
					logger.Error("入库失败", zap.Error(err), zap.String("path", fullPath))

					failedCount++

					if _, err = h.authIngestLogService.Create(ctx.GetContext(),
						req.PlanId, autoingest.LogLevelError,
						fmt.Sprintf("新增入库失败：%s, 错误信息：%s", fullPath, err.Error()),
					); err != nil {
						logger.Error("创建入库日志失败", zap.Error(err))
					}

					continue
				}

				taskReq := &topic.FileScanFileRequest{
					FileId: id,
					Deep:   true,
				}

				body, _ := json.Marshal(taskReq)
				if err = h.taskEngine.PushMessage(
					ctx.GetContext().
						WithValue(consts.CtxKeyFullPath, fullPath).
						WithValue(consts.CtxKeyInvokeHandlerName, "入库执行器"),
					taskReq.Topic(), body); err != nil {
					logger.Error("下发文件扫描任务失败", zap.Error(err))
				}

				if _, err = h.authIngestLogService.Create(ctx.GetContext(),
					req.PlanId, autoingest.LogLevelInfo,
					fmt.Sprintf("新增入库：%s", fullPath),
				); err != nil {
					logger.Error("创建入库日志失败", zap.Error(err))
				}

				addCount++
				time.Sleep(50 * time.Millisecond)
			}
		}

		return nil
	}
}
