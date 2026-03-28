package utils

import (
	"net/url"
	"path"
	"regexp"
	"strings"
)

func SplitPath(path string) ([]string, error) {
	// 按照路径分隔符分割路径
	parts := strings.Split(path, "/")

	// 创建结果切片
	result := make([]string, 0)

	for _, part := range parts {
		// 跳过空字符串
		if part == "" {
			continue
		}

		// 对特殊字符进行转义
		escaped, err := url.PathUnescape(part)
		if err != nil {
			return nil, err
		}

		result = append(result, escaped)
	}

	return result, nil
}

// CheckIsPath 检查是不是一个路径 必须 / 开头
func CheckIsPath(path string) bool {
	if path == "" {
		return false
	}
	// 检查是否以 / 开头
	if !strings.HasPrefix(path, "/") {
		return false
	}
	// 检查路径是否包含非法字符
	if strings.Contains(path, "\\") {
		return false
	}

	return true
}

func PathEscape(elem ...string) string {
	if len(elem) == 0 {
		return ""
	}

	// 先拼接路径
	joined := path.Join(elem...)

	// 检查是否需要前导斜杠
	needsLeadingSlash := len(elem) > 0 && strings.HasPrefix(elem[0], "/")

	// 分割并转义每个部分
	ss := strings.Split(joined, "/")

	var ns = make([]string, 0, len(ss))

	for _, s := range ss {
		if s != "" { // 过滤空字符串
			ns = append(ns, url.PathEscape(s))
		}
	}

	result := path.Join(ns...)

	// 添加前导斜杠（如果需要）
	if needsLeadingSlash && !strings.HasPrefix(result, "/") {
		result = "/" + result
	}

	return result
}

// SanitizeFileName 清理 Windows 非法文件名字符
func SanitizeFileName(name string) string {
	// Windows 禁用字符: < > : " | ? * \ /
	invalidChars := regexp.MustCompile(`[<>:"|?*\\/]`)

	// 替换为下划线或其他安全字符
	cleaned := invalidChars.ReplaceAllString(name, "_")

	// 也可以替换为中文字符
	cleaned = strings.ReplaceAll(cleaned, "|", "丨") // 用中文竖线替代

	return strings.TrimSpace(cleaned)
}
