package utils

import (
	"crypto/rand"
	"math/big"
)

const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
)

// GenerateString 生成随机字符串
func GenerateString(length int) string {
	password := make([]byte, length)

	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// 如果加密随机数生成失败，回退到时间种子
			return generateStringFallback(length)
		}

		password[i] = charset[num.Int64()]
	}

	return string(password)
}

// generateStringFallback 回退的生成方法
func generateStringFallback(length int) string {
	// 使用当前时间的纳秒作为种子
	seed := make([]byte, 8)
	_, _ = rand.Read(seed)

	password := make([]byte, length)
	for i := range password {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		password[i] = charset[num.Int64()]
	}

	return string(password)
}
