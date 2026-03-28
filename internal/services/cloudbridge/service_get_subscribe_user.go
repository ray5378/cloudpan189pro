package cloudbridge

import (
	"errors"
	"fmt"
	"time"

	"github.com/xxcheng123/cloudpan189-interface/client"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
	"go.uber.org/zap"
)

type SubscribeUserInfo struct {
	ID     int64  `json:"id"`
	UserId string `json:"userId"`
	Name   string `json:"name"`
}

func (s *service) GetSubscribeUserInfo(ctx context.Context, userId string) (*SubscribeUserInfo, error) {
	if info, err := s.getClient(ctx).SubscribeGetUser(ctx, userId); err != nil {
		ctx.Error("查询订阅用户信息失败", zap.String("userId", userId), zap.Error(err))

		return nil, err
	} else {
		return &SubscribeUserInfo{
			ID:     info.Data.Id,
			UserId: info.Data.UserId,
			Name:   info.Data.Name,
		}, nil
	}
}

type SubscribeUserShareResourceOption struct {
	PageNum  int
	PageSize int
	FileName string
}

type SubscribeUserShareResourceOptionFunc func(opt *SubscribeUserShareResourceOption)

type ShareResourceInfo struct {
	UserId     string    `json:"userId"`
	Name       string    `json:"name"`
	IsFolder   bool      `json:"isFolder"`
	AccessCode string    `json:"accessCode"`
	ShareId    int64     `json:"shareId"`
	ID         string    `json:"id"`
	ShareTime  time.Time `json:"shareTime"`
	IsTop      int       `json:"isTop"`
}

func (s *service) GetSubscribeUserShareResource(ctx context.Context, userId string, opts ...SubscribeUserShareResourceOptionFunc) ([]*ShareResourceInfo, int64, error) {
	option := &SubscribeUserShareResourceOption{
		PageNum:  1,
		PageSize: 30,
	}

	for _, opt := range opts {
		opt(option)
	}

	resp, err := s.getClient(ctx).GetUpResourceShare(ctx, userId, int64(option.PageNum), int64(option.PageSize), func(req *client.GetUpResourceShareRequest) {
		req.FileName = option.FileName
	})
	if err != nil {
		ctx.Error("获取订阅号下级分享失败", zap.String("user_id", userId), zap.Error(err))

		return nil, 0, err
	}

	if resp == nil || resp.Data == nil {
		ctx.Error("获取订阅号下级分享返回为空", zap.String("user_id", userId))

		return nil, 0, errors.New("获取订阅号下级分享返回为空")
	}

	list := make([]*ShareResourceInfo, 0)

	for _, item := range resp.Data.FileList {
		shareTime, _ := time.Parse(time.DateTime, item.ShareDate)

		list = append(list, &ShareResourceInfo{
			UserId:     userId,
			Name:       utils.SanitizeFileName(item.Name),
			IsFolder:   item.Folder == 1,
			AccessCode: item.AccessURL,
			ShareId:    item.ShareId,
			ID:         fmt.Sprint(item.Id),
			ShareTime:  shareTime,
			IsTop:      item.IsTop,
		})
	}

	return list, resp.Data.Count, nil
}
