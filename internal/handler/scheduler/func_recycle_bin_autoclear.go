package scheduler

import (
	"strings"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/tickstep/cloudpan189-api/cloudpan"
	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	appsessionSvi "github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	casrestoreSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrestore"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"go.uber.org/zap"
)

type RecycleBinAutoClearScheduler struct {
	appSessionService appsessionSvi.Service
	ctx               appctx.Context
	cancel            appctx.CancelFunc
	running           bool
	lastRunKey        string
}

func NewRecycleBinAutoClearScheduler(appSessionService appsessionSvi.Service, _ cloudtokenSvi.Service) Scheduler {
	return &RecycleBinAutoClearScheduler{appSessionService: appSessionService}
}

func (s *RecycleBinAutoClearScheduler) Start(ctx appctx.Context) error {
	if s.running {
		return ErrSchedulerRunning
	}
	s.ctx, s.cancel = appctx.WithCancel(ctx)
	s.running = true
	gopool.Go(func() {
		for s.doJob() {
		}
	})
	return nil
}

func (s *RecycleBinAutoClearScheduler) Stop() {
	if !s.running {
		return
	}
	s.cancel()
	s.running = false
}

func (s *RecycleBinAutoClearScheduler) doJob() bool {
	ctx := s.ctx
	logger := ctx.Logger
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if !shared.SettingAddition.RecycleBinAutoClearEnabled {
				continue
			}
			now := time.Now()
			runKey := now.Format("2006-01-02")
			if s.lastRunKey == runKey {
				continue
			}
			target := strings.TrimSpace(shared.SettingAddition.RecycleBinAutoClearTime)
			if target == "" {
				target = "03:30"
			}
			if now.Format("15:04") != target {
				continue
			}
			if err := s.clearPersonRecycle(ctx); err != nil {
				logger.Error("定时清空个人回收站失败", zap.Error(err))
			} else {
				logger.Info("定时清空个人回收站完成")
			}
			if err := s.clearFamilyRecycle(ctx); err != nil {
				logger.Error("定时清空家庭回收站失败", zap.Error(err))
			} else {
				logger.Info("定时清空家庭回收站完成")
			}
			s.lastRunKey = runKey
		}
	}
}

func (s *RecycleBinAutoClearScheduler) clearPersonRecycle(ctx appctx.Context) error {
	tokenID := shared.SettingAddition.CasPersonTargetTokenId
	if tokenID <= 0 {
		return nil
	}
	session, err := s.appSessionService.GetByTokenID(ctx, tokenID)
	if err != nil {
		return err
	}
	panClient := cloudpan.NewPanClient(cloudpan.WebLoginToken{}, session.Token)
	if apiErr := panClient.RecycleClear(0); apiErr != nil {
		return apiErr
	}
	return nil
}

func (s *RecycleBinAutoClearScheduler) clearFamilyRecycle(ctx appctx.Context) error {
	tokenID := shared.SettingAddition.CasFamilyTargetTokenId
	familyID := strings.TrimSpace(shared.SettingAddition.CasFamilyTargetFamilyId)
	if tokenID <= 0 || familyID == "" {
		return nil
	}
	session, err := s.appSessionService.GetByTokenID(ctx, tokenID)
	if err != nil {
		return err
	}
	accessToken := strings.TrimSpace(session.Token.AccessToken)
	if accessToken == "" {
		return nil
	}
	return casrestoreSvi.ClearFamilyRecycleByAccessToken(accessToken, familyID)
}
