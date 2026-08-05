package worklog

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"
)

type OutputOptions struct {
	Format string
	Fields []string
	Writer io.Writer
}

func Render(value any, options OutputOptions) error {
	if options.Writer == nil {
		options.Writer = io.Discard
	}
	format := strings.ToLower(strings.TrimSpace(options.Format))
	if format == "" {
		format = "json"
	}
	switch format {
	case "json":
		encoder := json.NewEncoder(options.Writer)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	case "jsonl":
		rows, err := rowsFromValue(value)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(options.Writer)
		encoder.SetEscapeHTML(false)
		for _, row := range rows {
			if err := encoder.Encode(row); err != nil {
				return err
			}
		}
		return nil
	case "table":
		rows, err := rowsFromValue(value)
		if err != nil {
			return err
		}
		return renderTable(options.Writer, rows, options.Fields)
	case "csv":
		rows, err := rowsFromValue(value)
		if err != nil {
			return err
		}
		return renderCSV(options.Writer, rows, options.Fields)
	default:
		return usageError("不支持的输出格式 %q（可选 json/jsonl/table/csv）", options.Format)
	}
}

func rowsFromValue(value any) ([]map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	if list, ok := raw.([]any); ok {
		rows := make([]map[string]any, 0, len(list))
		for _, item := range list {
			if row, ok := item.(map[string]any); ok {
				rows = append(rows, row)
			} else {
				rows = append(rows, map[string]any{"value": item})
			}
		}
		return rows, nil
	}
	if row, ok := raw.(map[string]any); ok {
		return []map[string]any{row}, nil
	}
	return []map[string]any{{"value": raw}}, nil
}

func renderTable(writer io.Writer, rows []map[string]any, selected []string) error {
	columns := chooseColumns(rows, selected)
	if len(columns) == 0 {
		_, err := fmt.Fprintln(writer, "(empty)")
		return err
	}
	widths := make([]int, len(columns))
	for i, column := range columns {
		widths[i] = runeWidth(column)
	}
	for _, row := range rows {
		for i, column := range columns {
			if width := runeWidth(cellText(row[column])); width > widths[i] {
				widths[i] = width
			}
		}
	}
	writeRow := func(row map[string]any) error {
		parts := make([]string, len(columns))
		for i, column := range columns {
			text := cellText(row[column])
			parts[i] = padRight(text, widths[i])
		}
		_, err := fmt.Fprintln(writer, strings.Join(parts, "  "))
		return err
	}
	if err := writeRow(headerMap(columns)); err != nil {
		return err
	}
	separator := make([]string, len(columns))
	for i := range columns {
		separator[i] = strings.Repeat("-", widths[i])
	}
	if _, err := fmt.Fprintln(writer, strings.Join(separator, "  ")); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writeRow(row); err != nil {
			return err
		}
	}
	return nil
}

func renderCSV(writer io.Writer, rows []map[string]any, selected []string) error {
	columns := chooseColumns(rows, selected)
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(columns); err != nil {
		return err
	}
	for _, row := range rows {
		values := make([]string, len(columns))
		for i, column := range columns {
			values[i] = cellText(row[column])
		}
		if err := csvWriter.Write(values); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func chooseColumns(rows []map[string]any, selected []string) []string {
	if len(selected) > 0 {
		return append([]string(nil), selected...)
	}
	seen := map[string]bool{}
	columns := []string{}
	for _, row := range rows {
		keys := make([]string, 0, len(row))
		for key := range row {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if !seen[key] {
				seen[key] = true
				columns = append(columns, key)
			}
		}
	}
	return columns
}

func headerMap(columns []string) map[string]any {
	row := map[string]any{}
	for _, column := range columns {
		row[column] = column
	}
	return row
}

func cellText(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	if reflect.ValueOf(value).Kind() == reflect.String {
		return reflect.ValueOf(value).String()
	}
	data, err := json.Marshal(value)
	if err == nil && string(data) != "null" {
		return string(data)
	}
	return fmt.Sprint(value)
}

func runeWidth(text string) int { return utf8.RuneCountInString(text) }

func padRight(text string, width int) string {
	return text + strings.Repeat(" ", maxInt(0, width-runeWidth(text)))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
