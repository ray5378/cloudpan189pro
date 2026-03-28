package group2file

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
)

// BatchBindFiles 批量绑定文件权限到用户组 先删除 再绑定
func (s *service) BatchBindFiles(ctx context.Context, groupId int64, fileIds []int64) error {
	// 删除所有分组
	if err := s.getDB(ctx).Where("group_id = ?", groupId).Delete(new(models.Group2File)).Error; err != nil {
		ctx.Error("删除分组失败", zap.Error(err), zap.Int64("groupId", groupId), zap.Int64s("fileIds", fileIds))

		return err
	}
	// 添加新的分组
	items := make([]models.Group2File, 0, len(fileIds))
	for _, fileId := range fileIds {
		items = append(items, models.Group2File{
			GroupId: groupId,
			FileId:  fileId,
		})
	}

	if err := s.getDB(ctx).Create(&items).Error; err != nil {
		ctx.Error("创建分组失败", zap.Error(err), zap.Int64("groupId", groupId))

		return err
	}

	ctx.Info("批量绑定文件权限成功", zap.Int64("groupId", groupId), zap.Int("fileCount", len(fileIds)))

	return nil
}
