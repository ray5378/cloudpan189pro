package casrecord

import (
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListRequest CAS 恢复记录列表查询请求。
type ListRequest struct {
	StorageID    int64                   `form:"storageId" binding:"omitempty"`
	MountPointID int64                   `form:"mountPointId" binding:"omitempty"`
	RestoreStatus models.CasRestoreStatus `form:"restoreStatus" binding:"omitempty"`
	CasFileName  string                  `form:"casFileName" binding:"omitempty"`

	BeginAt time.Time `form:"beginAt" binding:"omitempty"`
	EndAt   time.Time `form:"endAt" binding:"omitempty"`

	CurrentPage int  `form:"currentPage,omitempty,default=1" binding:"omitempty,min=1"`
	PageSize    int  `form:"pageSize,omitempty,default=10" binding:"omitempty,min=1"`
	NoPaginate  bool `form:"-"`

	AscList  []string `form:"-"`
	DescList []string `form:"-"`
}

func (s *service) List(ctx context.Context, req *ListRequest) ([]*models.CasMediaRecord, error) {
	query := s.getListQuery(ctx, req)
	if len(req.AscList) > 0 {
		for _, k := range req.AscList {
			query = query.Order(clause.OrderByColumn{Column: clause.Column{Name: k}, Desc: false})
		}
	} else {
		query = query.Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: true})
	}
	for _, k := range req.DescList {
		query = query.Order(clause.OrderByColumn{Column: clause.Column{Name: k}, Desc: true})
	}
	if !req.NoPaginate && req.CurrentPage > 0 && req.PageSize > 0 {
		query = query.Offset((req.CurrentPage - 1) * req.PageSize).Limit(req.PageSize)
	}

	list := make([]*models.CasMediaRecord, 0)
	return list, query.Find(&list).Error
}

func (s *service) Count(ctx context.Context, req *ListRequest) (int64, error) {
	var count int64
	err := s.getListQuery(ctx, req).Count(&count).Error
	return count, err
}

func (s *service) getListQuery(ctx context.Context, req *ListRequest) *gorm.DB {
	query := s.getDB(ctx)
	if req == nil {
		return query
	}
	if req.StorageID > 0 {
		query = query.Where("storage_id = ?", req.StorageID)
	}
	if req.MountPointID > 0 {
		query = query.Where("mount_point_id = ?", req.MountPointID)
	}
	if req.RestoreStatus != "" {
		query = query.Where("restore_status = ?", req.RestoreStatus)
	}
	if req.CasFileName != "" {
		query = query.Where("cas_file_name LIKE ?", "%"+req.CasFileName+"%")
	}
	if !req.BeginAt.IsZero() {
		query = query.Where("created_at >= ?", req.BeginAt)
	}
	if !req.EndAt.IsZero() {
		query = query.Where("created_at <= ?", req.EndAt)
	}
	return query
}
