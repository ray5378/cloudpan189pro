package bootstrap

import (
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"gorm.io/gorm"
)

func migrateDB(db *gorm.DB) (err error) {
	return db.AutoMigrate(
		new(models.Setting),
		new(models.User),
		new(models.UserGroup),
		new(models.Group2File),
		new(models.VirtualFile),
		new(models.MediaFile),
		new(models.FileTaskLog),
		new(models.CloudToken),
		new(models.MountPoint),
		new(models.AutoIngestLog),
		new(models.AutoIngestPlan),
		new(models.LoginLog),
		new(models.MediaConfig),
		new(models.CasMediaRecord),
	)
}

const (
	defaultWebTitle = "天翼订阅小站"
)

func initSetting(db *gorm.DB) error {
	var count int64

	db.Model(new(models.Setting)).Count(&count)

	if count > 0 {
		return nil
	}

	setting := &models.Setting{
		Title:      defaultWebTitle,
		EnableAuth: true,
		SaltKey:    utils.GenerateString(16),
		Addition: models.SettingAddition{
			Keep: utils.GenerateString(16), // 维持结构
		},
	}

	return db.Create(setting).Error
}
