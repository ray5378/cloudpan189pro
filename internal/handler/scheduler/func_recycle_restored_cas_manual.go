package scheduler

import (
	"math/rand"
	"time"

	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	casrecordSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrecord"
	appsessionSvi "github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
)

// RunCASFallbackRecycleOnce 立即执行一次 CAS 恢复文件兜底扫描清理。
func RunCASFallbackRecycleOnce(ctx appctx.Context, casRecordService casrecordSvi.Service, appSessionService appsessionSvi.Service, cloudTokenService cloudtokenSvi.Service) error {
	retentionHours := shared.SettingAddition.CasRestoreRetentionHours
	if retentionHours <= 0 {
		return nil
	}
	s := &RecycleRestoredCASScheduler{
		casRecordService:  casRecordService,
		appSessionService: appSessionService,
		cloudTokenService: cloudTokenService,
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	cutoff := time.Now().Add(-time.Duration(retentionHours) * time.Hour)
	return s.runFallbackRecycleSweep(ctx, cutoff)
}
