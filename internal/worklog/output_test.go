package worklog

import (
	"strings"
	"testing"
)

func TestRuneWidthCountsFullWidthRunesAsTwoColumns(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"张三", 4},
		{"内部平台建设", 12},
		{"客户 A 交付", 11}, // 4 个全角字 + 2 空格 + 1 个 ASCII
		{"（全角）", 8},
	}
	for _, tc := range cases {
		if got := runeWidth(tc.text); got != tc.want {
			t.Fatalf("runeWidth(%q) = %d, want %d", tc.text, got, tc.want)
		}
	}
}

func TestRenderTableAlignsColumnsWithCJKText(t *testing.T) {
	rows := []map[string]any{
		{"id": "246", "name": "内部平台建设"},
		{"id": "312", "name": "客户 A 交付"},
	}
	var buffer strings.Builder
	if err := Render(rows, OutputOptions{Format: "table", Writer: &buffer}); err != nil {
		t.Fatalf("Render 失败: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buffer.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("期望 4 行输出，实际 %d 行: %q", len(lines), lines)
	}
	// 每行 name 列都应从同一列开始，否则终端里表格会错位。
	want := runeWidth(lines[0][:strings.Index(lines[0], "name")])
	for _, line := range lines[1:] {
		fields := strings.SplitN(line, "  ", 2)
		if len(fields) != 2 {
			t.Fatalf("行 %q 没有列分隔", line)
		}
		if got := runeWidth(fields[0]) + 2; got != want {
			t.Fatalf("行 %q 第二列起始列 = %d, want %d", line, got, want)
		}
	}
}

func TestRenderTableTrimsTrailingPadding(t *testing.T) {
	rows := []map[string]any{
		{"name": "内部平台建设"},
		{"name": "短"},
	}
	var buffer strings.Builder
	if err := Render(rows, OutputOptions{Format: "table", Writer: &buffer}); err != nil {
		t.Fatalf("Render 失败: %v", err)
	}
	for _, line := range strings.Split(strings.TrimRight(buffer.String(), "\n"), "\n") {
		if strings.HasSuffix(line, " ") {
			t.Fatalf("行 %q 存在行尾空格", line)
		}
	}
}
