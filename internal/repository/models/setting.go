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
	EnableAuth  bool            `gorm:"column:enable_auth;type:tinyint(1);default:1" json:"enableAuth"` // 是否启用鉴权 1 启用 0 不启用
	SaltKey     string          `gorm:"column:salt_key;type:varchar(255);not null" json:"-"`
	BaseURL     string          `gorm:"column:base_url;type:varchar(255);not null;default:''" json:"baseURL"` // base url
	Initialized bool            `gorm:"column:initialized;type:tinyint(1);default:0" json:"initialized"`      // 是否初始化完成
	Addition    SettingAddition `gorm:"column:addition;type:json" json:"addition" swaggertype:"object"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime;type:datetime;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;autoUpdateTime;type:datetime;default:CURRENT_TIMESTAMP;on update:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (s *Setting) TableName() string {
	return "setting"
}

// AfterFind GORM hook: 在查询完成后注入默认值
func (s *Setting) AfterFind(tx *gorm.DB) (err error) {
	s.Addition.applyDefaults()

	return nil
}

// SettingAddition 系统附加配置
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
}

// applyDefaults 统一填充默认值，确保零值时也能获得期望配置
func (sa *SettingAddition) applyDefaults() {
	if sa.MultipleStreamThreadCount <= 0 {
		sa.MultipleStreamThreadCount = 4
	}

	if sa.MultipleStreamChunkSize <= 0 {
		sa.MultipleStreamChunkSize = 4 * 1024 * 1024 // 4MiB
	}

	if sa.TaskThreadCount <= 0 {
		sa.TaskThreadCount = 1
	}

	// Defaults for external creation
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
		sa.AutoDeleteInvalidStorageKeywords = "资源不存在|文件不存在|目录不存在|分享已失效|分享不存在"
	}
}

// Value 实现 driver.Valuer 接口 - 将结构体转换为数据库值
func (sa SettingAddition) Value() (driver.Value, error) {
	if sa == (SettingAddition{}) {
		return nil, nil
	}

	return json.Marshal(sa)
}

// Scan 实现 sql.Scanner 接口 - 从数据库值转换为结构体
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
