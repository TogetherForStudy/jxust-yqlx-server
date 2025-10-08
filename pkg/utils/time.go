package utils

import (
	"errors"
	"time"
)

// ParseDateTime 统一的时间解析方法，支持多种格式
func ParseDateTime(dateStr string) (*time.Time, error) {
	if dateStr == "" {
		return nil, nil
	}

	// 获取中国时区
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600) // 备用：固定UTC+8
	}

	// 支持的时间格式
	formats := []string{
		"2006-01-02 15:04:05",       // 完整日期时间
		"2006-01-02 15:04",          // 当前使用的格式
		"2006-01-02T15:04:05Z",      // ISO 8601 UTC
		"2006-01-02T15:04:05+08:00", // ISO 8601 带时区
		time.RFC3339,                // 标准ISO格式
		time.RFC3339Nano,            // 带纳秒的ISO格式
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, dateStr, loc); err == nil {
			return &t, nil
		}
		// 对于UTC格式，直接解析然后转换到本地时区
		if t, err := time.Parse(format, dateStr); err == nil {
			localTime := t.In(loc)
			return &localTime, nil
		}
	}

	return nil, errors.New("不支持的时间格式")
}

// GetLocalTime 获取当前的本地时区时间
func GetLocalTime() time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600) // 备用：固定UTC+8
	}
	return time.Now().In(loc)
}

// GetChinaLocation 获取中国时区
func GetChinaLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600) // 备用：固定UTC+8
	}
	return loc
}

// FormatDateTime 格式化时间为字符串
func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// FormatDateTimeShort 格式化时间为短格式字符串
func FormatDateTimeShort(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// CalculateDaysDiff 计算两个日期之间的天数差异
func CalculateDaysDiff(from, to time.Time) int {
	loc := GetChinaLocation()
	fromDate := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)
	toDate := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, loc)
	return int(toDate.Sub(fromDate).Hours() / 24)
}
