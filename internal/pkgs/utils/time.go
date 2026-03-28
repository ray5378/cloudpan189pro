package utils

import (
	"fmt"
	"strings"
	"time"
)

// 时间单位常量
const (
	Day   = 24 * time.Hour
	Week  = 7 * Day
	Month = 30 * Day  // 近似值，用于显示
	Year  = 365 * Day // 近似值，用于显示
)

// DurationUnit 时间单位定义
type DurationUnit struct {
	Duration time.Duration
	Name     string
}

// 预定义的时间单位（从大到小排序）
var durationUnits = []DurationUnit{
	{Year, "年"},
	{Month, "月"},
	{Week, "周"},
	{Day, "天"},
	{time.Hour, "小时"},
	{time.Minute, "分"},
	{time.Second, "秒"},
}

// FormatDuration 格式化时间间隔
// 例如：1年2月3天4小时5分6秒
// 自动省略为0的单位，如：2天3小时
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0秒"
	}

	if d < 0 {
		return "-" + FormatDuration(-d)
	}

	var parts []string

	remaining := d

	for _, unit := range durationUnits {
		if remaining >= unit.Duration {
			count := remaining / unit.Duration
			remaining %= unit.Duration
			parts = append(parts, fmt.Sprintf("%d%s", count, unit.Name))
		}
	}

	if len(parts) == 0 {
		return "0秒"
	}

	return strings.Join(parts, "")
}

// FormatDurationSimple 简化格式的时间间隔
// 只显示最大的两个有意义的单位
// 例如：1年2月、3天4小时、5分6秒
func FormatDurationSimple(d time.Duration) string {
	if d == 0 {
		return "0秒"
	}

	if d < 0 {
		return "-" + FormatDurationSimple(-d)
	}

	var parts []string

	remaining := d
	maxParts := 2

	for _, unit := range durationUnits {
		if remaining >= unit.Duration && len(parts) < maxParts {
			count := remaining / unit.Duration
			remaining %= unit.Duration
			parts = append(parts, fmt.Sprintf("%d%s", count, unit.Name))
		}
	}

	if len(parts) == 0 {
		return "0秒"
	}

	return strings.Join(parts, "")
}

// FormatDurationHuman 人性化的时间间隔格式
// 根据时间长度自动选择合适的精度
func FormatDurationHuman(d time.Duration) string {
	if d == 0 {
		return "刚刚"
	}

	if d < 0 {
		return FormatDurationHuman(-d) + "前"
	}

	// 小于1分钟
	if d < time.Minute {
		return "刚刚"
	}

	// 小于1小时，显示分钟
	if d < time.Hour {
		minutes := d / time.Minute

		return fmt.Sprintf("%d分钟", minutes)
	}

	// 小于1天，显示小时和分钟
	if d < Day {
		hours := d / time.Hour
		minutes := (d % time.Hour) / time.Minute

		if minutes == 0 {
			return fmt.Sprintf("%d小时", hours)
		}

		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	}

	// 小于1周，显示天和小时
	if d < Week {
		days := d / Day
		hours := (d % Day) / time.Hour

		if hours == 0 {
			return fmt.Sprintf("%d天", days)
		}

		return fmt.Sprintf("%d天%d小时", days, hours)
	}

	// 大于1周，使用简化格式
	return FormatDurationSimple(d)
}
