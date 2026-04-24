package scheduler

import (
	"fmt"

	"github.com/tickstep/cloudpan189-api/cloudpan"
	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
)

func (s *RecycleRestoredCASScheduler) newPanClientBySession(ctx appctx.Context, session *appsession.Session) (*cloudpan.PanClient, error) {
	_ = ctx
	if session == nil {
		return nil, fmt.Errorf("session为空")
	}
	return cloudpan.NewPanClient(cloudpan.WebLoginToken{}, session.Token), nil
}
