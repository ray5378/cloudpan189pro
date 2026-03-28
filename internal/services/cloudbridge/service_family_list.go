package cloudbridge

import (
	"github.com/xxcheng123/cloudpan189-interface/client"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"go.uber.org/zap"
)

func (s *service) FamilyList(ctx context.Context, token client.AuthToken) (*GetFamilyListResponse, error) {
	resp, err := client.New().
		WithClient(ctx.HTTPClient()).
		WithToken(token).
		GetFamilyList(ctx)
	if err != nil {
		ctx.Error("获取家庭云列表失败", zap.Error(err))

		return nil, err
	}

	return resp, nil
}
