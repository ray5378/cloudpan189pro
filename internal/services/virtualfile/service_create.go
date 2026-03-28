package virtualfile

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BatchCreateHook func(ctx context.Context, result *gorm.DB, files []*models.VirtualFile)

func (s *service) BatchCreate(ctx context.Context, parentId int64, files []*models.VirtualFile, hooks ...BatchCreateHook) (int64, error) {
	ctx.Debug("批量创建文件", zap.Int64("parent_id", parentId), zap.Int("file_count", len(files)))

	// 检查 pid
	if parentId < 0 {
		return 0, errors.New("parent_id is invalid")
	}

	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.Name)
	}

	existNames, err := s.queryHasExist(ctx, parentId, names)
	if err != nil {
		return 0, err
	}

	existNameSet := lo.SliceToMap(existNames, func(item string) (string, bool) {
		return item, true
	})

	for _, file := range files {
		file.ParentId = parentId
		file.Name = utils.SanitizeFileName(file.Name)

		if exist, ok := existNameSet[file.Name]; ok && exist {
			ctx.Debug("文件已存在 - 重命名", zap.String("file_name", file.Name), zap.String("new_name", fmt.Sprintf("%s(%s)", file.Name, file.Rev)), zap.String("cloud_file_id", file.CloudId))

			file.Name = fmt.Sprintf("%s(%s)", file.Name, file.Rev)
		} else {
			existNameSet[file.Name] = true
		}
	}

	result := s.withLock(ctx, func(db *gorm.DB) *gorm.DB {
		return db.CreateInBatches(files, 1000)
	})

	for _, hook := range hooks {
		hook(ctx, result, files)
	}

	return result.RowsAffected, result.Error
}

func (s *service) Create(ctx context.Context, parentId int64, file *models.VirtualFile) (int64, error) {
	ctx.Debug("创建文件", zap.Int64("parent_id", parentId), zap.String("file_name", file.Name))

	// 检查 pid
	if parentId < 0 {
		return 0, errors.New("parent_id is invalid")
	}

	existNames, err := s.queryHasExist(ctx, parentId, []string{file.Name})
	if err != nil {
		return 0, err
	}

	if len(existNames) > 0 {
		file.Name = fmt.Sprintf("%s(%s)", file.Name, file.Rev)
	}

	file.ParentId = parentId
	file.Name = utils.SanitizeFileName(file.Name)

	result := s.withLock(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Create(file)
	})

	return file.ID, result.Error
}

func (s *service) CreateTop(ctx context.Context, parentId int64, file *models.VirtualFile) (int64, error) {
	ctx.Debug("创建挂载点", zap.Int64("parent_id", parentId), zap.String("file_name", file.Name))

	id, err := s.Create(ctx, parentId, file)
	if err != nil {
		return 0, err
	}

	if err = s.Update(ctx, id, []utils.Field{utils.WithField("top_id", file.ID)}); err != nil {
		ctx.Error("回写 top_id 失败", zap.Int64("file_id", id), zap.Error(err))

		return 0, err
	}

	return id, nil
}

func (s *service) queryHasExist(ctx context.Context, pid int64, names []string) ([]string, error) {
	if len(names) <= 0 {
		return nil, nil
	}

	var existNames = make([]string, 0)

	if err := s.getDB(ctx).Where("parent_id = ? AND name IN (?)", pid, names).Select("name").Find(&existNames).Error; err != nil {
		return nil, err
	}

	return existNames, nil
}
