package worklog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWeekBoundsAndBuildWeekEntries(t *testing.T) {
	day := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC) // Wednesday
	start, end, err := WeekBounds(day, 0)
	if err != nil {
		t.Fatal(err)
	}
	if start.Format("2006-01-02") != "2026-08-03" || end.Format("2006-01-02") != "2026-08-09" {
		t.Fatalf("unexpected Monday week: %s ~ %s", start, end)
	}
	sundayStart, sundayEnd, err := WeekBounds(day, 6)
	if err != nil {
		t.Fatal(err)
	}
	if sundayStart.Format("2006-01-02") != "2026-08-02" || sundayEnd.Format("2006-01-02") != "2026-08-08" {
		t.Fatalf("unexpected Sunday week: %s ~ %s", sundayStart, sundayEnd)
	}
	_, _, entries, err := BuildWeekEntries(day, "246", "联调", 8, 1, false)
	if err != nil || len(entries) != 5 {
		t.Fatalf("expected five workdays, got %d, err=%v", len(entries), err)
	}
	_, _, entries, err = BuildWeekEntries(day, "246", "联调", 8, 1, true)
	if err != nil || len(entries) != 7 {
		t.Fatalf("expected seven days, got %d, err=%v", len(entries), err)
	}
}

func TestParseEntriesAliasesAndTrailingJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "entries.json")
	if err := os.WriteFile(path, []byte(`{"entries":[{"rq":"2026-07-28T00:00:00","gs":"8","工作内容":"开发","完成进度":"0.5","xmmc":"246"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	entries, err := ParseEntriesFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ProjectID != "246" || entries[0].Progress != 0.5 {
		t.Fatalf("unexpected entry: %+v", entries)
	}
	if err := os.WriteFile(path, []byte(`[] []`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseEntriesFile(path); err == nil {
		t.Fatal("expected trailing JSON error")
	}
}
