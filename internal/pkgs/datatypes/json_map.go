package datatypes

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// JSONMap defined JSON data type, need to implements driver.Valuer, sql.Scanner interface
type JSONMap map[string]interface{}

// Value return json value, implement driver.Valuer interface
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}

	ba, err := m.MarshalJSON()

	return string(ba), err
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (m *JSONMap) Scan(val interface{}) error {
	if val == nil {
		*m = make(JSONMap)

		return nil
	}

	var ba []byte

	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", val))
	}

	t := map[string]interface{}{}
	rd := bytes.NewReader(ba)
	decoder := json.NewDecoder(rd)
	decoder.UseNumber()
	err := decoder.Decode(&t)
	*m = t

	return err
}

// MarshalJSON to output non base64 encoded []byte
func (m JSONMap) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}

	t := (map[string]interface{})(m)

	return json.Marshal(t)
}

// UnmarshalJSON to deserialize []byte
func (m *JSONMap) UnmarshalJSON(b []byte) error {
	t := map[string]interface{}{}
	err := json.Unmarshal(b, &t)
	*m = JSONMap(t)

	return err
}

func (m *JSONMap) Unmarshal(v any) error {
	bs, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(bs, v)
}

// GormDataType gorm common data type
func (m JSONMap) GormDataType() string {
	return "jsonmap"
}

// GormDBDataType gorm db data type
func (JSONMap) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Name() {
	case "sqlite":
		return "JSON"
	case "mysql":
		return "JSON"
	case "postgres":
		return "JSONB"
	case "sqlserver":
		return "NVARCHAR(MAX)"
	}

	return ""
}

func (jm JSONMap) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	data, _ := jm.MarshalJSON()

	switch db.Name() {
	case "mysql":
		if v, ok := db.Dialector.(*mysql.Dialector); ok && !strings.Contains(v.ServerVersion, "MariaDB") {
			return gorm.Expr("CAST(? AS JSON)", string(data))
		}
	}

	return gorm.Expr("?", string(data))
}

// String 获取字符串类型的值
func (jm JSONMap) String(key string) (string, bool) {
	if jm == nil {
		return "", false
	}

	value, exists := jm[key]
	if !exists {
		return "", false
	}

	switch v := value.(type) {
	case string:
		return v, true
	case json.Number:
		return string(v), true
	default:
		// 尝试转换为字符串
		if str := fmt.Sprintf("%v", v); str != "" {
			return str, true
		}

		return "", false
	}
}

// Int 获取整数类型的值
func (jm JSONMap) Int(key string) (int, bool) {
	if jm == nil {
		return 0, false
	}

	value, exists := jm[key]
	if !exists {
		return 0, false
	}

	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i), true
		}

		return 0, false
	case string:
		var result int
		if n, err := fmt.Sscanf(v, "%d", &result); err == nil && n == 1 {
			return result, true
		}

		return 0, false
	default:
		return 0, false
	}
}

// Int64 获取int64类型的值
func (jm JSONMap) Int64(key string) (int64, bool) {
	if jm == nil {
		return 0, false
	}

	value, exists := jm[key]
	if !exists {
		return 0, false
	}

	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i, true
		}

		return 0, false
	case string:
		var result int64
		if n, err := fmt.Sscanf(v, "%d", &result); err == nil && n == 1 {
			return result, true
		}

		return 0, false
	default:
		return 0, false
	}
}

// Bool 获取布尔类型的值
func (jm JSONMap) Bool(key string) (bool, bool) {
	if jm == nil {
		return false, false
	}

	value, exists := jm[key]
	if !exists {
		return false, false
	}

	switch v := value.(type) {
	case bool:
		return v, true
	case int:
		return v != 0, true
	case int64:
		return v != 0, true
	case float64:
		return v != 0, true
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i != 0, true
		}

		if f, err := v.Float64(); err == nil {
			return f != 0, true
		}

		return false, false
	case string:
		switch strings.ToLower(v) {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off", "":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

// Set 设置键值对
func (jm JSONMap) Set(key string, value interface{}) {
	if jm == nil {
		return
	}

	jm[key] = value
}

// FromStruct 将带有 json 标签的结构体转换为 JSONMap（使用 UseNumber 保留数字精度）
func FromStruct(v interface{}) (JSONMap, error) {
	if v == nil {
		return nil, nil
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()

	if err := dec.Decode(&m); err != nil {
		return nil, err
	}

	return JSONMap(m), nil
}
