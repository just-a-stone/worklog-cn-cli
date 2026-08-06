package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/just-a-stone/worklog-cn-cli/internal/worklog"
	"github.com/spf13/cobra"
)

type reportArgs struct {
	file           string
	week           string
	date           string
	hours          float64
	content        string
	progress       float64
	project        string
	includeWeekend bool
	start          string
	end            string
	title          string
	src            string
	noLinkage      bool
	nextWeekPlan   string
	dumpPayload    string
	iConfirm       bool
}

func newDryRunCommand(options *cliOptions) *cobra.Command {
	args := &reportArgs{}
	cmd := &cobra.Command{Use: "dry-run", Aliases: []string{"dry-run-report"}, Short: "组装一周 Timesheet payload，不提交", Args: cobra.NoArgs, RunE: func(_ *cobra.Command, _ []string) error { return runReport(options, args, false) }}
	addReportFlags(cmd, args)
	return cmd
}

func newSubmitCommand(options *cliOptions) *cobra.Command {
	args := &reportArgs{}
	cmd := &cobra.Command{Use: "submit", Aliases: []string{"submit-report"}, Short: "预览或提交一周 Timesheet", Args: cobra.NoArgs, RunE: func(_ *cobra.Command, _ []string) error {
		return runReport(options, args, args.iConfirm || options.yes)
	}}
	addReportFlags(cmd, args)
	cmd.Flags().BoolVar(&args.iConfirm, "i-confirm", false, "确认写库；缺少时只 dry-run")
	return cmd
}

func addReportFlags(cmd *cobra.Command, args *reportArgs) {
	cmd.Flags().StringVarP(&args.file, "file", "f", "", "entries JSON 文件")
	cmd.Flags().StringVar(&args.week, "week", "", "周内任一天 YYYY-MM-DD")
	cmd.Flags().StringVar(&args.date, "date", "", "同 --week（兼容旧参数）")
	cmd.Flags().Float64Var(&args.hours, "hours", 8, "按周模式每天工时")
	cmd.Flags().StringVarP(&args.content, "content", "c", "", "按周模式每天工作内容")
	cmd.Flags().Float64Var(&args.progress, "progress", 1, "完成进度 0~1")
	cmd.Flags().StringVarP(&args.project, "project", "p", "", "默认项目 id")
	cmd.Flags().BoolVar(&args.includeWeekend, "include-weekend", false, "按周模式包含周六日")
	cmd.Flags().StringVar(&args.start, "start", "", "主表开始日期")
	cmd.Flags().StringVar(&args.end, "end", "", "主表结束日期")
	cmd.Flags().StringVar(&args.title, "title", "", "流程标题")
	cmd.Flags().StringVar(&args.src, "src", "submit", "提交类型：submit 或 save")
	cmd.Flags().BoolVar(&args.noLinkage, "no-linkage", false, "跳过服务端联动")
	cmd.Flags().StringVar(&args.nextWeekPlan, "next-week-plan", "", "下周计划")
	cmd.Flags().StringVar(&args.dumpPayload, "dump-payload", "", "写出完整 payload JSON")
}

func runReport(options *cliOptions, args *reportArgs, commit bool) error {
	cfg, err := options.config()
	if err != nil {
		return err
	}
	defaultProject := cfg.DefaultProject
	if args.project != "" {
		defaultProject = args.project
	}
	entries, startDate, endDate, err := loadReportEntries(args, defaultProject)
	if err != nil {
		return err
	}
	if err := validateEntries(entries, defaultProject); err != nil {
		return err
	}
	if options.readonly && commit {
		return worklog.RefusedError("--readonly 拒绝 submit")
	}
	cookie, err := worklog.ResolveCookie(options.cookie, cfg)
	if err != nil {
		return err
	}
	cfg.Cookie = cookie
	client, err := worklog.NewClient(cfg)
	if err != nil {
		return err
	}
	if args.src != "submit" && args.src != "save" {
		return worklog.UsageError("src 只能是 submit 或 save")
	}
	prepared, err := client.PrepareSubmit(context.Background(), entries, defaultProject, startDate, endDate, args.title, args.src, !args.noLinkage, args.nextWeekPlan, time.Now())
	if err != nil {
		return err
	}
	if args.dumpPayload != "" {
		data, _ := json.MarshalIndent(prepared.Payload, "", "  ")
		if err := os.WriteFile(args.dumpPayload, append(data, '\n'), 0600); err != nil {
			return worklog.ConfigError("写入 payload 失败: %v", err)
		}
	}
	result := map[string]any{
		"dry_run":             !commit,
		"submitted":           false,
		"i_confirm":           commit,
		"week":                map[string]any{"start": dateString(startDate), "end": dateString(endDate), "row_count": len(entries)},
		"context":             prepared.Context,
		"entries":             prepared.Entries,
		"payload_summary":     prepared.PayloadSummary,
		"payload_field_count": len(prepared.Payload),
		"dump_payload":        args.dumpPayload,
	}
	if commit {
		if err := client.CheckSecondAuth(context.Background(), prepared.Form, args.src); err != nil {
			return err
		}
		apiResult, err := client.RequestOperation(context.Background(), prepared.Payload)
		if err != nil {
			return err
		}
		if failed, message := operationFailed(apiResult); failed {
			return worklog.RemoteError("requestOperation 失败: %s", message)
		}
		result["submitted"] = true
		result["api_result"] = apiResult
	}
	if options.outputFormat() == "table" {
		return renderReportTable(options, result, prepared)
	}
	return options.output(result)
}

func loadReportEntries(args *reportArgs, defaultProject string) ([]worklog.WorkEntry, *time.Time, *time.Time, error) {
	var startDate, endDate *time.Time
	if args.start != "" {
		value, err := parseDate(args.start)
		if err != nil {
			return nil, nil, nil, worklog.UsageError("start 日期无效")
		}
		startDate = &value
	}
	if args.end != "" {
		value, err := parseDate(args.end)
		if err != nil {
			return nil, nil, nil, worklog.UsageError("end 日期无效")
		}
		endDate = &value
	}
	if args.file != "" {
		entries, err := worklog.ParseEntriesFile(args.file)
		if err != nil {
			return nil, nil, nil, err
		}
		if startDate == nil && endDate == nil && len(entries) > 0 {
			earliest := entries[0].WorkDate
			for _, entry := range entries[1:] {
				if entry.WorkDate.Before(earliest) {
					earliest = entry.WorkDate
				}
			}
			start, end, err := worklog.WeekBounds(earliest, 0)
			if err != nil {
				return nil, nil, nil, err
			}
			startDate, endDate = &start, &end
		}
		return entries, startDate, endDate, nil
	}
	weekText := args.week
	if weekText == "" {
		weekText = args.date
	}
	if weekText == "" {
		return nil, nil, nil, worklog.UsageError("需要 --week/--date，或 -f entries.json")
	}
	if strings.TrimSpace(defaultProject) == "" && strings.TrimSpace(args.project) == "" {
		return nil, nil, nil, worklog.UsageError("需要 --project 或 ECOLOGY_DEFAULT_PROJECT")
	}
	if strings.TrimSpace(args.content) == "" {
		return nil, nil, nil, worklog.UsageError("按周模式需要 --content")
	}
	weekOf, err := parseDate(weekText)
	if err != nil {
		return nil, nil, nil, worklog.UsageError("week 日期无效")
	}
	project := args.project
	if project == "" {
		project = defaultProject
	}
	start, end, entries, err := worklog.BuildWeekEntries(weekOf, project, args.content, args.hours, args.progress, args.includeWeekend)
	if err != nil {
		return nil, nil, nil, err
	}
	if startDate == nil {
		startDate = &start
	}
	if endDate == nil {
		endDate = &end
	}
	return entries, startDate, endDate, nil
}

func validateEntries(entries []worklog.WorkEntry, defaultProject string) error {
	if len(entries) == 0 {
		return worklog.UsageError("无 entries")
	}
	errors := []string{}
	for _, entry := range entries {
		if strings.TrimSpace(entry.ProjectID) == "" && strings.TrimSpace(defaultProject) == "" {
			errors = append(errors, entry.WorkDate.Format("2006-01-02")+" 缺少 project_id")
		}
		if entry.Hours <= 0 {
			errors = append(errors, entry.WorkDate.Format("2006-01-02")+" hours 必须 > 0")
		}
		if strings.TrimSpace(entry.Content) == "" {
			errors = append(errors, entry.WorkDate.Format("2006-01-02")+" content 为空")
		}
		if entry.Progress < 0 || entry.Progress > 1 {
			errors = append(errors, entry.WorkDate.Format("2006-01-02")+" progress 必须在 0 到 1 之间")
		}
	}
	if len(errors) > 0 {
		return worklog.UsageError("参数错误: %s", strings.Join(errors, "; "))
	}
	return nil
}

func renderReportTable(options *cliOptions, result map[string]any, prepared worklog.PreparedSubmit) error {
	rows := make([]map[string]any, 0, len(prepared.PayloadSummary["rows"].([]map[string]any)))
	if raw, ok := prepared.PayloadSummary["rows"].([]map[string]any); ok {
		rows = raw
	}
	if len(rows) == 0 {
		rows = []map[string]any{{"dry_run": result["dry_run"], "submitted": result["submitted"], "payload_field_count": result["payload_field_count"]}}
	}
	return options.output(rows)
}

func operationFailed(value any) (bool, string) {
	object, ok := value.(map[string]any)
	if !ok {
		return false, ""
	}
	success, ok := object["success"].(bool)
	if ok && !success {
		message := fmt.Sprint(object["message"])
		if message == "<nil>" || message == "" {
			message = fmt.Sprint(object["msg"])
		}
		return true, message
	}
	if text, ok := object["success"].(string); ok && strings.EqualFold(text, "false") {
		return true, fmt.Sprint(object["message"])
	}
	return false, ""
}

func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", strings.TrimSpace(value), time.Local)
}

func dateString(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Format("2006-01-02")
}
