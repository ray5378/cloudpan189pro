package scheduler

import (
	"math/rand"
	"time"

	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	casrecordSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrecord"
	appsessionSvi "github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
)

// RunCASFallbackRecycleOnce 立即执行一次 CAS 恢复文件清理。
// 注意：这里复用原有“按恢复记录驱动”的回收主链，而不是目录兜底扫描链。
func RunCASFallbackRecycleOnce(ctx appctx.Context, casRecordService casrecordSvi.Service, appSessionService appsessionSvi.Service, cloudTokenService cloudtokenSvi.Service) error {
	s := &RecycleRestoredCASScheduler{
		casRecordService:  casRecordService,
		appSessionService: appSessionService,
		cloudTokenService: cloudTokenService,
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	return s.runRecordedRecycleOnce(ctx, time.Now())
}
