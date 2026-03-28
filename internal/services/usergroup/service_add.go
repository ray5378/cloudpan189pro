package usergroup

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
)

type AddRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255" example:"管理员组"` // 用户组名称，长度1-255位
}

type AddResponse struct {
	ID int64 `json:"id" example:"1001"` // 新创建用户组的ID
}

func (s *service) Add(ctx context.Context, req *AddRequest) (resp *AddResponse, err error) {
	group := models.UserGroup{
		Name: req.Name,
	}

	if err = s.getDB(ctx).Create(&group).Error; err != nil {
		ctx.Error("用户组创建失败", zap.String("name", req.Name), zap.Error(err))

		return nil, err
	}

	return &AddResponse{
		ID: group.ID,
	}, nil
}
