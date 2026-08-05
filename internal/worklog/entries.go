package worklog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

func parseString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func parseFloat(value any) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case json.Number:
		return typed.Float64()
	case string:
		return strconv.ParseFloat(strings.TrimSpace(typed), 64)
	default:
		return 0, fmt.Errorf("unsupported number %T", value)
	}
}

func WorkEntryFromMap(raw map[string]any) (WorkEntry, error) {
	dateRaw := firstValue(raw, "date", "work_date", "rq")
	if dateRaw == nil || strings.TrimSpace(parseString(dateRaw)) == "" {
		return WorkEntry{}, usageError("entry 缺少 date")
	}
	dateText := parseString(dateRaw)
	if len(dateText) >= 10 {
		dateText = dateText[:10]
	}
	dateValue, err := time.Parse("2006-01-02", dateText)
	if err != nil {
		return WorkEntry{}, usageError("entry 日期无效: %s", dateText)
	}
	hoursRaw := firstValue(raw, "hours", "gs", "工时")
	if hoursRaw == nil {
		return WorkEntry{}, usageError("entry %s 缺少 hours", dateText)
	}
	hours, err := parseFloat(hoursRaw)
	if err != nil {
		return WorkEntry{}, usageError("entry %s hours 无效", dateText)
	}
	content := parseString(firstValue(raw, "content", "gznr", "工作内容"))
	if strings.TrimSpace(content) == "" {
		return WorkEntry{}, usageError("entry %s 缺少 content", dateText)
	}
	progress := 1.0
	if rawProgress := firstValue(raw, "progress", "wcjd", "完成进度"); rawProgress != nil {
		progress, err = parseFloat(rawProgress)
		if err != nil {
			return WorkEntry{}, usageError("entry %s progress 无效", dateText)
		}
	}
	if progress < 0 || progress > 1 {
		return WorkEntry{}, usageError("entry %s progress 必须在 0 到 1 之间", dateText)
	}
	return WorkEntry{
		WorkDate:    dateValue,
		Hours:       hours,
		Content:     content,
		Progress:    progress,
		ProjectID:   optionalString(firstValue(raw, "project_id", "xmmc")),
		ProjectName: optionalString(firstValue(raw, "project_name", "xmmc_name")),
		ProjectCode: optionalString(firstValue(raw, "project_code", "xmbh")),
		ManagerID:   optionalString(firstValue(raw, "manager_id", "xmjl")),
	}, nil
}

func firstValue(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := values[key]; ok && value != nil && parseString(value) != "" {
			return value
		}
	}
	return nil
}

func optionalString(value any) string {
	if value == nil {
		return ""
	}
	text := strings.TrimSpace(parseString(value))
	return text
}

func ParseEntriesFile(path string) ([]WorkEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, usageError("entries 文件包含多个 JSON 值")
		}
		return nil, err
	}
	var list []any
	switch typed := raw.(type) {
	case []any:
		list = typed
	case map[string]any:
		entries, ok := typed["entries"].([]any)
		if !ok {
			return nil, usageError("entries 文件必须是 JSON 数组或 {entries: [...]}")
		}
		list = entries
	default:
		return nil, usageError("entries 文件必须是 JSON 数组或 {entries: [...]}")
	}
	entries := make([]WorkEntry, 0, len(list))
	for i, item := range list {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, usageError("entry %d 必须是对象", i)
		}
		entry, err := WorkEntryFromMap(object)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func Daterange(start, end time.Time) []time.Time {
	days := []time.Time{}
	for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
		days = append(days, current)
	}
	return days
}

func WeekBounds(day time.Time, weekStart int) (time.Time, time.Time, error) {
	if weekStart != 0 && weekStart != 6 {
		return time.Time{}, time.Time{}, usageError("week_start 仅支持 0(周一) 或 6(周日)")
	}
	day = dateOnly(day)
	offset := (int(day.Weekday()) + 6) % 7 // Monday=0
	if weekStart == 6 {
		offset = int(day.Weekday())
	}
	start := day.AddDate(0, 0, -offset)
	return start, start.AddDate(0, 0, 6), nil
}

func BuildWeekEntries(weekOf time.Time, projectID, content string, hours, progress float64, includeWeekend bool) (time.Time, time.Time, []WorkEntry, error) {
	start, end, err := WeekBounds(weekOf, 0)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}
	if strings.TrimSpace(projectID) == "" {
		return time.Time{}, time.Time{}, nil, usageError("project id 不能为空")
	}
	if strings.TrimSpace(content) == "" {
		return time.Time{}, time.Time{}, nil, usageError("content 不能为空")
	}
	if progress < 0 || progress > 1 {
		return time.Time{}, time.Time{}, nil, usageError("progress 必须在 0 到 1 之间")
	}
	entries := make([]WorkEntry, 0, 7)
	for _, day := range Daterange(start, end) {
		if !includeWeekend && (day.Weekday() == time.Saturday || day.Weekday() == time.Sunday) {
			continue
		}
		entries = append(entries, WorkEntry{WorkDate: day, Hours: hours, Content: content, Progress: progress, ProjectID: projectID})
	}
	return start, end, entries, nil
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
