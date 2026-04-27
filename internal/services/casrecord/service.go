package casrecord

import (
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"gorm.io/gorm"
)

type Service interface {
	Create(ctx context.Context, record *models.CasMediaRecord) (int64, error)
	Query(ctx context.Context, id int64) (*models.CasMediaRecord, error)
	QueryByStorageAndCasFileID(ctx context.Context, storageID int64, casFileID string) (*models.CasMediaRecord, error)
	QueryByRestoredFileID(ctx context.Context, restoredFileID string) (*models.CasMediaRecord, error)
	ListDueRecycle(ctx context.Context, now time.Time, limit int) ([]*models.CasMediaRecord, error)
	Update(ctx context.Context, id int64, updates map[string]any) error
	DeleteByCasFilePath(ctx context.Context, casFilePath string) error
}

type service struct {
	svc bootstrap.ServiceContext
}

func NewService(svc bootstrap.ServiceContext) Service {
	return &service{svc: svc}
}

func (s *service) getDB(ctx context.Context) *gorm.DB {
	return s.svc.GetDB(ctx).Model(new(models.CasMediaRecord))
}
