package cloudbridge

import (
	"time"

	"github.com/xxcheng123/cloudpan189-interface/client"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"go.uber.org/zap"
)

type ShareInfo struct {
	Name       string    `json:"name"`
	IsFolder   bool      `json:"isFolder"`
	AccessCode string    `json:"accessCode"`
	ShareId    int64     `json:"shareId"`
	ID         string    `json:"id"`
	ShareTime  time.Time `json:"shareTime"`
}

func (s *service) GetShareInfo(ctx context.Context, shareCode string, accessCode string) (*ShareInfo, error) {
	info, err := s.getClient(ctx).GetShareInfo(ctx, shareCode, func(gsir *client.GetShareInfoRequest) {
		gsir.AccessCode = accessCode
	})
	if err != nil {
		ctx.Error("获取分享详情失败", zap.String("shareCode", shareCode), zap.Error(err))

		return nil, err
	}

	shareTime, _ := time.Parse(time.DateTime, info.FileCreateDate)

	return &ShareInfo{
		Name:       info.FileName,
		IsFolder:   info.IsFolder,
		AccessCode: info.AccessCode,
		ShareId:    info.ShareId,
		ID:         string(info.FileId),
		ShareTime:  shareTime,
	}, nil
}
