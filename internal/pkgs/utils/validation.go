package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// 验证标签到中文错误信息的映射
var validationTagMap = map[string]string{
	"required": "不能为空",
	"min":      "长度不能少于%s位",
	"max":      "长度不能超过%s位",
	"url":      "格式不正确，请输入有效的URL",
}

// TranslateValidationError 将验证错误转换为中文
func TranslateValidationError(err error) string {
	var messages []string

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, fieldError := range validationErrors {
			fieldName := fieldError.Field()
			tag := fieldError.Tag()
			param := fieldError.Param()

			// 获取验证错误的中文描述
			var errorMsg string

			if template, exists := validationTagMap[tag]; exists {
				if param != "" {
					errorMsg = fmt.Sprintf(template, param)
				} else {
					errorMsg = template
				}
			} else {
				errorMsg = "格式不正确"
			}

			messages = append(messages, fmt.Sprintf("%s%s", fieldName, errorMsg))
		}
	}

	if len(messages) == 0 {
		return "参数验证失败"
	}

	return strings.Join(messages, "；")
}
