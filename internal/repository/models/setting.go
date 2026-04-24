package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/ptr"
	"gorm.io/gorm"
)

type Setting struct {
	ID          int64           `gorm:"primaryKey" json:"id"`
	Title       string          `gorm:"column:title;type:varchar(255);not null" json:"title"`
	EnableAuth  bool            `gorm:"column:enable_auth;type:tinyint(1);default:1" json:"enableAuth"`
	SaltKey     string          `gorm:"column:salt_key;type:varchar(255);not null" json:"-"`
	BaseURL     string          `gorm:"column:base_url;type:varchar(255);not null;default:''" json:"baseURL"`
	Initialized bool            `gorm:"column:initialized;type:tinyint(1);default:0" json:"initialized"`
	Addition    SettingAddition `gorm:"column:addition;type:json" json:"addition" swaggertype:"object"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime;type:datetime;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;autoUpdateTime;type:datetime;default:CURRENT_TIMESTAMP;on update:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (s *Setting) TableName() string { return "setting" }

func (s *Setting) AfterFind(tx *gorm.DB) (err error) {
	s.Addition.applyDefaults()
	return nil
}

type SettingAddition struct {
	Keep                      string `json:"-"`
	LocalProxy                bool   `json:"localProxy"`
	MultipleStream            bool   `json:"multipleStream"`
	MultipleStreamThreadCount int    `json:"multipleStreamThreadCount"`
	MultipleStreamChunkSize   int64  `json:"multipleStreamChunkSize"`
	TaskThreadCount           int    `json:"taskThreadCount"`

	ExternalAPIKey             string `json:"externalApiKey"`
	DefaultTokenId             int64  `json:"defaultTokenId"`
	ExternalAutoRefreshEnabled *bool  `json:"externalAutoRefreshEnabled"`
	ExternalRefreshIntervalMin int    `json:"externalRefreshIntervalMin"`
	ExternalAutoRefreshDays    int    `json:"externalAutoRefreshDays"`

	PersistentCheckEnabled bool   `json:"persistentCheckEnabled"`
	PersistentCheckDay     int    `json:"persistentCheckDay"`
	PersistentCheckTime    string `json:"persistentCheckTime"`

	AutoDeleteInvalidStorageEnabled  bool   `json:"autoDeleteInvalidStorageEnabled"`
	AutoDeleteInvalidStorageKeywords string `json:"autoDeleteInvalidStorageKeywords"`

	CasTargetEnabled            bool   `json:"casTargetEnabled"`
	CasTargetType               string `json:"casTargetType"`
	CasPersonTargetTokenId      int64  `json:"casPersonTargetTokenId"`
	CasPersonTargetFolderId     string `json:"casPersonTargetFolderId"`
	CasPersonAccessPath         string `json:"casPersonAccessPath"`
	CasFamilyTargetTokenId      int64  `json:"casFamilyTargetTokenId"`
	CasFamilyTargetFamilyId     string `json:"casFamilyTargetFamilyId"`
	CasFamilyTargetFolderId     string `json:"casFamilyTargetFolderId"`
	CasFamilyAccessPath         string `json:"casFamilyAccessPath"`
	CasRestoreRetentionHours    int    `json:"casRestoreRetentionHours"`
	RecycleBinAutoClearEnabled  bool   `json:"recycleBinAutoClearEnabled"`
	RecycleBinAutoClearTime     string `json:"recycleBinAutoClearTime"`
	LocalCASAutoScanEnabled     bool   `json:"localCasAutoScanEnabled"`
	LocalCASAutoScanIntervalMin int    `json:"localCasAutoScanIntervalMin"`
	CasAutoCollectEnabled       bool   `json:"casAutoCollectEnabled"`
	CasAutoCollectPreservePath  bool   `json:"casAutoCollectPreservePath"`
}

func (sa *SettingAddition) applyDefaults() {
	if sa.MultipleStreamThreadCount <= 0 {
		sa.MultipleStreamThreadCount = 4
	}
	if sa.MultipleStreamChunkSize <= 0 {
		sa.MultipleStreamChunkSize = 4 * 1024 * 1024
	}
	if sa.TaskThreadCount <= 0 {
		sa.TaskThreadCount = 1
	}
	if sa.ExternalAutoRefreshEnabled == nil {
		sa.ExternalAutoRefreshEnabled = ptr.Of(true)
	}
	if sa.ExternalRefreshIntervalMin <= 0 {
		sa.ExternalRefreshIntervalMin = 60
	}
	if sa.ExternalAutoRefreshDays <= 0 {
		sa.ExternalAutoRefreshDays = 60
	}
	if sa.PersistentCheckDay <= 0 || sa.PersistentCheckDay > 28 {
		sa.PersistentCheckDay = 1
	}
	if sa.PersistentCheckTime == "" {
		sa.PersistentCheckTime = "03:00"
	}
	if sa.AutoDeleteInvalidStorageKeywords == "" {
		sa.AutoDeleteInvalidStorageKeywords = "分享审核|分享审核不通过|没有找到分享信息|页面不存在|文件或文件夹不存在|分享平铺目录未找到"
	}
	if sa.CasTargetType == "" {
		sa.CasTargetType = "person"
	}
	if sa.RecycleBinAutoClearTime == "" {
		sa.RecycleBinAutoClearTime = "03:30"
	}
	if sa.LocalCASAutoScanIntervalMin <= 0 {
		sa.LocalCASAutoScanIntervalMin = 10
	}
	if !sa.CasAutoCollectPreservePath {
		sa.CasAutoCollectPreservePath = true
	}
}

func (sa SettingAddition) Value() (driver.Value, error) {
	if sa == (SettingAddition{}) {
		return nil, nil
	}
	return json.Marshal(sa)
}

func (sa *SettingAddition) Scan(value interface{}) error {
	if value == nil {
		*sa = SettingAddition{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("cannot scan SettingAddition from non-string/[]byte value")
	}
	return json.Unmarshal(bytes, sa)
}
