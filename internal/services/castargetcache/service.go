package castargetcache

import (
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"gorm.io/gorm"
)

// Service 提供 CAS 目标目录缓存操作。
//
// 重要原则（不要随意改）：
// - 这里的缓存只应该被“开启自动刷新的存储”使用。
// - 不要把这个 service 直接推广成全局目录缓存层。
// - 缓存内容必须来自目标云盘目录实际列表，而不是本地臆造/历史成功记录。
// - 本地缓存只负责提速和去重，不能替代云盘真实状态。
type Service interface {
	Exists(ctx context.Context, targetTokenID int64, targetFolderID, fileName string) (bool, error)
	Upsert(ctx context.Context, item *models.CasTargetDirCache) error
	RefreshDir(ctx context.Context, targetTokenID int64, targetFolderID string, items []*models.CasTargetDirCache) error
	NeedsRefresh(ctx context.Context, targetTokenID int64, targetFolderID string, ttl time.Duration) (bool, error)
	IsEmpty(ctx context.Context) (bool, error)
	ListDistinctDirs(ctx context.Context) ([]*models.CasTargetDirCache, error)
}

type service struct {
	svc bootstrap.ServiceContext
}

func NewService(svc bootstrap.ServiceContext) Service {
	return &service{svc: svc}
}

func (s *service) getDB(ctx context.Context) *gorm.DB {
	return s.svc.GetDB(ctx).Model(new(models.CasTargetDirCache))
}
