package models

import "time"

type CasRestoreStatus string

const (
	CasRestoreStatusPending   CasRestoreStatus = "pending"
	CasRestoreStatusRestoring CasRestoreStatus = "restoring"
	CasRestoreStatusRestored  CasRestoreStatus = "restored"
	CasRestoreStatusFailed    CasRestoreStatus = "failed"
	CasRestoreStatusRecycling CasRestoreStatus = "recycling"
	CasRestoreStatusRecycled  CasRestoreStatus = "recycled"
)

type CasMediaRecord struct {
	ID               int64            `gorm:"primaryKey" json:"id"`
	StorageID        int64            `gorm:"column:storage_id;type:bigint;not null;default:0;uniqueIndex:uk_storage_cas" json:"storageId"`
	MountPointID     int64            `gorm:"column:mount_point_id;type:bigint;not null;default:0" json:"mountPointId"`
	CasFileID        string           `gorm:"column:cas_file_id;type:varchar(128);not null;uniqueIndex:uk_storage_cas" json:"casFileId"`
	CasFileName      string           `gorm:"column:cas_file_name;type:varchar(1024);not null" json:"casFileName"`
	CasFilePath      string           `gorm:"column:cas_file_path;type:varchar(2048);not null;default:''" json:"casFilePath"`
	SourceParentID   string           `gorm:"column:source_parent_id;type:varchar(128);not null;default:''" json:"sourceParentId"`
	RestoredParentID string           `gorm:"column:restored_parent_id;type:varchar(128);not null;default:''" json:"restoredParentId"`
	OriginalFileName string           `gorm:"column:original_file_name;type:varchar(1024);not null;default:''" json:"originalFileName"`
	OriginalFileSize int64            `gorm:"column:original_file_size;type:bigint;not null;default:0" json:"originalFileSize"`
	FileMD5          string           `gorm:"column:file_md5;type:varchar(64);not null;default:''" json:"fileMd5"`
	SliceMD5         string           `gorm:"column:slice_md5;type:varchar(64);not null;default:''" json:"sliceMd5"`
	StrmRelativePath string           `gorm:"column:strm_relative_path;type:varchar(2048);not null;default:''" json:"strmRelativePath"`
	RestoredFileID   string           `gorm:"column:restored_file_id;type:varchar(128);not null;default:''" json:"restoredFileId"`
	RestoredFileName string           `gorm:"column:restored_file_name;type:varchar(1024);not null;default:''" json:"restoredFileName"`
	RestoreStatus    CasRestoreStatus `gorm:"column:restore_status;type:varchar(32);not null;default:'pending';index:idx_cas_restore_status" json:"restoreStatus"`
	LastAccessAt     *time.Time       `gorm:"column:last_access_at;type:datetime" json:"lastAccessAt"`
	RestoredAt       *time.Time       `gorm:"column:restored_at;type:datetime" json:"restoredAt"`
	RecycleAfterAt   *time.Time       `gorm:"column:recycle_after_at;type:datetime;index:idx_cas_recycle_after_at" json:"recycleAfterAt"`
	LastError        string           `gorm:"column:last_error;type:text;not null;default:''" json:"lastError"`
	CreatedAt        time.Time        `gorm:"column:created_at;autoCreateTime;type:datetime;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt        time.Time        `gorm:"column:updated_at;autoUpdateTime;type:datetime;default:CURRENT_TIMESTAMP;on update:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (m *CasMediaRecord) TableName() string {
	return "cas_media_records"
}
