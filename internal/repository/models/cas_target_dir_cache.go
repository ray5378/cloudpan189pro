package models

import "time"

// CasTargetDirCache 是“目标云盘目录文件名”的本地缓存。
//
// 重要原则（不要随意改）：
// 1. 这个表缓存的数据源必须来自目标云盘目录本身，本地数据库只是缓存，不是真相来源。
// 2. 这个表只服务于“开启了自动刷新”的存储对应的自动转存去重。
// 3. 不要把缓存范围扩展到所有存储/所有手动转存，否则会把本地缓存错误地升级成全局事实来源。
// 4. 手动刷新是否使用这张表，必须服从上面的范围约束；不要为了省事改成全局共用缓存。
//
// 简单说：
// - 真相来自云盘
// - 本地只是镜像
// - 只缓存开启自动刷新的存储所涉及的目标目录
// - 不能把这张表改成“全系统所有转存目录”的全量缓存
//
// 后续如果要调整策略，请先确认不会破坏以上边界。
type CasTargetDirCache struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	TargetTokenID  int64     `gorm:"column:target_token_id;type:bigint;not null;default:0;uniqueIndex:uk_target_dir_name" json:"targetTokenId"`
	TargetFolderID string    `gorm:"column:target_folder_id;type:varchar(128);not null;default:'';uniqueIndex:uk_target_dir_name;index:idx_target_dir_refresh" json:"targetFolderId"`
	FileName       string    `gorm:"column:file_name;type:varchar(1024);not null;default:'';uniqueIndex:uk_target_dir_name" json:"fileName"`
	IsDir          bool      `gorm:"column:is_dir;type:boolean;not null;default:false" json:"isDir"`
	RefreshedAt    time.Time `gorm:"column:refreshed_at;type:datetime;index:idx_target_dir_refresh" json:"refreshedAt"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime;type:datetime;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime;type:datetime;default:CURRENT_TIMESTAMP;on update:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (m *CasTargetDirCache) TableName() string {
	return "cas_target_dir_caches"
}
