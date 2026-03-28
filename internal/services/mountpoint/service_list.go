package mountpoint

import (
	"fmt"
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ListRequest struct {
	TokenId           *int64 `form:"tokenId" binding:"omitempty" example:"1"`                               // 云盘令牌ID，可选
	CurrentPage       int    `form:"currentPage,omitempty,default=1" binding:"omitempty,min=1" example:"1"` // 当前页码，默认为1
	PageSize          int    `form:"pageSize,omitempty,default=10" binding:"omitempty,min=1" example:"10"`  // 每页大小，默认为10
	NoPaginate        bool   `form:"noPaginate" binding:"omitempty" example:"false"`                        // 是否不分页，默认false
	Name              string `form:"name" binding:"omitempty" example:"挂载点名称"`                              // 挂载点名称模糊搜索，可选
	FullPath          string `form:"fullPath" binding:"omitempty" example:"/path/to/mount"`                 // 完整路径模糊搜索，可选
	FileId            *int64 `form:"fileId" binding:"omitempty" example:"1"`                                // 文件ID
	EnableAutoRefresh *bool  `form:"enableAutoRefresh" binding:"omitempty" example:"true"`                  // 自动刷新
	LastState         string `form:"lastState" binding:"omitempty" example:"成功"`                            // 按状态筛选：成功、失败等

	// 新增：按字段排序与按最新任务状态筛选
	SortBy        string `form:"sortBy" binding:"omitempty" example:"fileCount"`
	SortOrder     string `form:"sortOrder" binding:"omitempty" example:"desc"`
	TaskLogStatus string `form:"taskLogStatus" binding:"omitempty" example:"completed"`
	FailureKind   string `form:"failureKind" binding:"omitempty,oneof=permanent transient" example:"permanent"`
}

func (s *service) List(ctx context.Context, req *ListRequest) (list []*models.MountPoint, err error) {
	query := s.getListQuery(ctx, req)

	// 应用分页
	if !req.NoPaginate {
		if req.CurrentPage <= 0 {
			req.CurrentPage = 1
		}

		if req.PageSize <= 0 {
			req.PageSize = 10
		}

		query = query.Offset((req.CurrentPage - 1) * req.PageSize).Limit(req.PageSize)
	}

	// 查询挂载点列表
	list = make([]*models.MountPoint, 0)
	if err = query.Find(&list).Error; err != nil {
		ctx.Error("查询挂载点列表失败", zap.Error(err))

		return nil, err
	}

	return list, nil
}

func (s *service) Count(ctx context.Context, req *ListRequest) (count int64, err error) {
	q := s.getListQuery(ctx, req)
	if err = q.Count(&count).Error; err != nil {
		ctx.Error("查询挂载点数量失败", zap.Error(err))

		return 0, err
	}

	return count, nil
}

func (s *service) getListQuery(ctx context.Context, req *ListRequest) *gorm.DB {
	query := s.getDB(ctx).Table("mount_points as m").Model(new(models.MountPoint))

	if req.Name != "" {
		query = query.Where("m.name LIKE ?", "%"+req.Name+"%")
	}

	if req.FullPath != "" {
		query = query.Where("m.full_path LIKE ?", "%"+req.FullPath+"%")
	}

	if req.FileId != nil {
		query = query.Where("m.file_id = ?", *req.FileId)
	}

	if req.TokenId != nil {
		query = query.Where("m.token_id = ?", *req.TokenId)
	}

	if req.EnableAutoRefresh != nil {
		query = query.Where("m.enable_auto_refresh = ?", *req.EnableAutoRefresh)
	}

	if req.LastState != "" {
		query = query.Where("m.last_state = ?", req.LastState)
	}

	// 最新任务日志状态筛选：JOIN 子查询 last(id 最大)
	needLastTaskJoin := req.TaskLogStatus != "" || req.FailureKind != ""
	if needLastTaskJoin {
		lastSub := s.svc.GetDB(ctx).Table("file_task_logs as fl1").
			Select("fl1.id, fl1.file_id, fl1.status").
			Joins("JOIN (SELECT file_id, MAX(id) AS max_id FROM file_task_logs GROUP BY file_id) t ON t.max_id = fl1.id")
		query = query.Joins("LEFT JOIN (?) AS last ON last.file_id = m.file_id", lastSub)
	}
	if req.TaskLogStatus != "" {
		query = query.Where("last.status = ?", req.TaskLogStatus)
	}
	if req.FailureKind != "" {
		query = query.Where("last.status = ?", models.StatusFailed)
		if req.FailureKind == "permanent" {
			query = query.Where("(m.enable_auto_refresh = ? OR m.auto_refresh_begin_at IS NULL OR datetime(m.auto_refresh_begin_at, '+' || m.auto_refresh_days || ' days') < ?)", false, time.Now())
		} else if req.FailureKind == "transient" {
			query = query.Where("m.enable_auto_refresh = ?").Where("m.auto_refresh_begin_at IS NOT NULL").Where("m.auto_refresh_begin_at <= ?", time.Now()).Where("datetime(m.auto_refresh_begin_at, '+' || m.auto_refresh_days || ' days') >= ?", time.Now())
		}
	}

	// 排序：fileCount 使用虚拟文件聚合子查询；其他列按 m.列 排
	sortOrder := "desc"
	if req.SortOrder == "asc" || req.SortOrder == "ASC" {
		sortOrder = "asc"
	}

	switch req.SortBy {
	case "fileCount":
		fcSub := s.svc.GetDB(ctx).Table("virtual_files").
			Select("top_id, COUNT(*) AS cnt").
			Where("top_id != id").
			Group("top_id")
		query = query.Joins("LEFT JOIN (?) AS fc ON fc.top_id = m.file_id", fcSub)
		query = query.Order(fmt.Sprintf("COALESCE(fc.cnt,0) %s", sortOrder)).Order("m.id asc")
	case "createdAt":
		query = query.Order(fmt.Sprintf("m.created_at %s", sortOrder)).Order("m.id asc")
	case "updatedAt":
		query = query.Order(fmt.Sprintf("m.updated_at %s", sortOrder)).Order("m.id asc")
	case "name":
		query = query.Order(fmt.Sprintf("m.name %s", sortOrder)).Order("m.id asc")
	default:
		query = query.Order("m.id desc")
	}

	return query
}
