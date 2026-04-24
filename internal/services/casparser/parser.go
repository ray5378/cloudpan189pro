package casparser

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type casPayload struct {
	Name        string `json:"name"`
	FileName    string `json:"fileName"`
	Size        int64  `json:"size"`
	FileSize    int64  `json:"fileSize"`
	MD5         string `json:"md5"`
	FileMD5     string `json:"fileMd5"`
	SliceMD5    string `json:"sliceMd5"`
	SliceMD5_   string `json:"slice_md5"`
	CreateTime  string `json:"createTime"`
	CreateTime2 string `json:"create_time"`
}

func IsCasFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(name)), ".cas")
}

func GetOriginalFileName(casFileName string, info *CasInfo) string {
	trimmed := strings.TrimSpace(casFileName)
	trimmed = strings.TrimSuffix(trimmed, ".cas")
	trimmed = strings.TrimSuffix(trimmed, ".CAS")
	if trimmed == "" {
		if info != nil && strings.TrimSpace(info.Name) != "" {
			return strings.TrimSpace(info.Name)
		}
		return strings.TrimSpace(casFileName)
	}

	ext := filepath.Ext(trimmed)
	if ext != "" && ext != "." {
		return trimmed
	}
	if info != nil {
		sourceExt := filepath.Ext(strings.TrimSpace(info.Name))
		if sourceExt != "" && sourceExt != "." {
			return trimmed + sourceExt
		}
	}
	return trimmed
}

func ParseCasContent(content []byte) (*CasInfo, error) {
	raw := strings.TrimSpace(string(bytes.TrimPrefix(content, []byte("\xef\xbb\xbf"))))
	if raw == "" {
		return nil, fmt.Errorf("CAS文件内容为空")
	}

	if strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}") {
		if info, err := parsePayload(raw); err == nil {
			return info, nil
		}
	}

	if decoded, err := tryBase64JSON(raw); err == nil {
		return decoded, nil
	}

	lines := strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == '\r' })
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "{") {
			if info, err := parsePayload(line); err == nil {
				return info, nil
			}
		}
		if decoded, err := tryBase64JSON(line); err == nil {
			return decoded, nil
		}
	}

	return nil, fmt.Errorf("CAS文件解析失败: 无法识别格式")
}

func tryBase64JSON(raw string) (*CasInfo, error) {
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	jsonStr := strings.TrimSpace(string(decoded))
	if !strings.HasPrefix(jsonStr, "{") {
		return nil, fmt.Errorf("decoded content is not json")
	}
	return parsePayload(jsonStr)
}

func parsePayload(jsonStr string) (*CasInfo, error) {
	var p casPayload
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return nil, err
	}
	info := &CasInfo{
		Name:       strings.TrimSpace(firstNonEmpty(p.Name, p.FileName)),
		Size:       firstPositive(p.Size, p.FileSize),
		MD5:        strings.ToUpper(strings.TrimSpace(firstNonEmpty(p.MD5, p.FileMD5))),
		SliceMD5:   strings.ToUpper(strings.TrimSpace(firstNonEmpty(p.SliceMD5, p.SliceMD5_))),
		CreateTime: strings.TrimSpace(firstNonEmpty(p.CreateTime2, p.CreateTime)),
	}
	if info.Name == "" {
		return nil, fmt.Errorf("CAS缺少文件名")
	}
	if info.Size < 0 {
		return nil, fmt.Errorf("CAS文件大小无效")
	}
	if info.MD5 == "" {
		return nil, fmt.Errorf("CAS缺少MD5")
	}
	if info.SliceMD5 == "" {
		return nil, fmt.Errorf("CAS缺少SliceMD5")
	}
	return info, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func firstPositive(values ...int64) int64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}
