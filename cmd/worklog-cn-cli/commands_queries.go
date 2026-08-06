package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/just-a-stone/worklog-cn-cli/internal/worklog"
	"github.com/spf13/cobra"
)

func newProjectsCommand(options *cliOptions) *cobra.Command {
	var pageSize int
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"list-projects"},
		Short:   "列出可选 Timesheet 项目",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			client, _, err := options.client(true)
			if err != nil {
				return err
			}
			form, err := client.LoadForm(context.Background(), worklog.WorkflowID)
			if err != nil {
				return err
			}
			projects, err := client.ListProjects(context.Background(), form, pageSize)
			if err != nil {
				return err
			}
			return options.output(projects)
		},
	}
	cmd.Flags().IntVar(&pageSize, "page-size", 50, "最多读取的项目数")
	return cmd
}

func newHistoryCommand(options *cliOptions) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:     "history",
		Aliases: []string{"list-history"},
		Short:   "列出 Timesheet 提报历史",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			client, _, err := options.client(true)
			if err != nil {
				return err
			}
			rows, err := client.ListHistory(context.Background(), limit)
			if err != nil {
				return err
			}
			return options.output(rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "最多返回条数")
	return cmd
}

func newViewCommand(options *cliOptions) *cobra.Command {
	return &cobra.Command{
		Use:     "view REQUEST_ID",
		Aliases: []string{"view-request"},
		Short:   "查看已提交工时单的主表与明细",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return worklog.UsageError("view 需要一个 requestid")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			client, _, err := options.client(true)
			if err != nil {
				return err
			}
			detail, err := client.GetRequestDetail(context.Background(), args[0])
			if err != nil {
				return err
			}
			if options.outputFormat() == "table" {
				return renderDetailTable(options, detail)
			}
			return options.output(detail)
		},
	}
}

func renderDetailTable(options *cliOptions, detail worklog.RequestDetail) error {
	rows := make([]map[string]any, 0, len(detail.Entries))
	for _, entry := range detail.Entries {
		rows = append(rows, map[string]any{"date": entry.WorkDate, "weekday": entry.Weekday, "hours": fmt.Sprintf("%.2f", entry.Hours), "progress": fmt.Sprintf("%.0f%%", entry.Progress*100), "project": entry.ProjectName, "project_id": entry.ProjectID, "content": entry.Content})
	}
	if len(rows) == 0 {
		rows = append(rows, map[string]any{"requestid": detail.RequestID, "title": detail.Title, "period": strings.TrimSpace(detail.StartDate + " ~ " + detail.EndDate)})
	}
	return options.output(rows)
}
