package worklog

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func BuildSubmitPayload(ctx FormContext, entries []WorkEntry, startDate, endDate *time.Time, lastTSEnd, requestName, src, nextWeekPlan string, now time.Time) (map[string]any, error) {
	if len(entries) == 0 {
		return nil, usageError("no work entries")
	}
	if now.IsZero() {
		now = time.Now()
	}
	entries = append([]WorkEntry(nil), entries...)
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].WorkDate.Before(entries[j].WorkDate) })
	start := dateOnly(entries[0].WorkDate)
	end := dateOnly(entries[len(entries)-1].WorkDate)
	if startDate != nil {
		start = dateOnly(*startDate)
	}
	if endDate != nil {
		end = dateOnly(*endDate)
	}
	if end.Before(start) {
		return nil, usageError("end_date before start_date")
	}
	duration := int(end.Sub(start).Hours()/24) + 1
	today := dateOnly(now).Format("2006-01-02")
	if requestName == "" {
		requestName = ctx.RequestNameDefault
	}
	if requestName == "" {
		identity := ctx.Username
		if identity == "" {
			identity = ctx.UserID
		}
		requestName = fmt.Sprintf("01.Timesheet提报-%s-%s", identity, today)
	}
	if src == "" {
		src = "submit"
	}
	payload := map[string]any{
		"formid":                     ctx.FormID,
		"f_weaver_belongto_userid":   ctx.UserID,
		"isWorkflowDoc":              "false",
		"f_weaver_belongto_usertype": 0,
		"nodetype":                   0,
		"method":                     "",
		"needoutprint":               "",
		"src":                        src,
		"isMultiDoc":                 "",
		"topage":                     "",
		ctx.SubmitTokenName:          ctx.SubmitToken,
		"workflowtype":               ctx.WorkflowType,
		"iscreate":                   1,
		"comemessage":                "",
		"remindTypes":                "",
		"rand":                       "",
		"requestid":                  -1,
		"linkageUUID":                ctx.LinkageUUID,
		"htmlfieldids":               "",
		"needwfback":                 1,
		"lastloginuserid":            ctx.UserID,
		"nodeid":                     ctx.NodeID,
		"workflowid":                 ctx.WorkflowID,
		"isbill":                     ctx.IsBill,
		"remark":                     "",
		"field-annexupload":          "",
		"signdocids":                 "",
		"signworkflowids":            "",
		"remarkLocation":             "",
		"annexdocids":                "",
		"annexdocinfos":              "",
		"handWrittenSign":            "",
		"isOdocRequest":              0,
		"enableIntervenor":           "",
		"linkageUnFinishedKey":       "",
		"remarkquote":                "",
		"actiontype":                 "requestOperation",
		"isFirstSubmit":              "",
		"field6436":                  "",
		"field6433":                  start.Format("2006-01-02"),
		"field6434":                  end.Format("2006-01-02"),
		"requestname":                requestName,
		"requestlevel":               0,
		"field6461":                  boolInt(start.Weekday() == time.Monday),
		"field6431":                  ctx.DeptID,
		"field6541":                  "",
		"field6541groupnum":          0,
		"field6432":                  today,
		"field6465":                  "",
		"field6462":                  1,
		"field6430":                  ctx.UserID,
		"field-10":                   "",
		"field6437":                  duration,
		"field6520":                  nextWeekPlan,
		"field6435":                  "",
		"field8800":                  lastTSEnd,
		"nodesnum0":                  len(entries),
		"indexnum0":                  len(entries),
		"submitdtlid0":               detailIDs(len(entries)),
		"deldtlid0":                  "",
		"nodesnum1":                  0,
		"indexnum1":                  0,
		"submitdtlid1":               "",
		"deldtlid1":                  "",
		"signatureAttributesStr":     ctx.SignatureAttributesStr,
		"signatureSecretKey":         ctx.SignatureSecretKey,
		"selectNextFlow":             0,
		"openDataVerify":             0,
		"wfTestStr":                  "",
	}
	required := []string{"field-9999", "field6433"}
	changed := []string{"field8800", "field6437", "field6462", "field6433", "field6434", "field6461", "field6430", "field6431", "field6432"}
	for i, entry := range entries {
		if strings.TrimSpace(entry.ProjectID) == "" {
			return nil, usageError("row %d missing project_id", i)
		}
		payload[fmt.Sprintf("field6448_%d", i)] = entry.ProjectID
		payload[fmt.Sprintf("field6448_%dname", i)] = entry.ProjectName
		payload[fmt.Sprintf("field6449_%d", i)] = entry.Content
		payload[fmt.Sprintf("field6459_%d", i)] = WeekdayCN[(int(entry.WorkDate.Weekday())+6)%7]
		payload[fmt.Sprintf("field6460_%d", i)] = dateOnly(entry.WorkDate).Format("2006-01-02")
		payload[fmt.Sprintf("field6463_%d", i)] = fmt.Sprintf("%.2f", entry.Hours)
		payload[fmt.Sprintf("field6464_%d", i)] = fmt.Sprintf("%.2f", entry.Progress)
		payload[fmt.Sprintf("field6467_%d", i)] = 0
		payload[fmt.Sprintf("field6477_%d", i)] = entry.ManagerID
		payload[fmt.Sprintf("field7110_%d", i)] = entry.ProjectCode
		required = append(required, fmt.Sprintf("field6448_%d", i), fmt.Sprintf("field6449_%d", i), fmt.Sprintf("field6463_%d", i), fmt.Sprintf("field6464_%d", i))
		changed = append(changed, fmt.Sprintf("field6448_%d", i), fmt.Sprintf("field6449_%d", i), fmt.Sprintf("field6459_%d", i), fmt.Sprintf("field6460_%d", i), fmt.Sprintf("field6463_%d", i), fmt.Sprintf("field6464_%d", i), fmt.Sprintf("field6477_%d", i), fmt.Sprintf("field7110_%d", i))
	}
	payload["verifyRequiredRange"] = strings.Join(required, ",") + ","
	payload["existChangeRange"] = strings.Join(uniqueStrings(changed), ",")
	payload["mainFieldUnEmptyCount"] = 9
	payload["detailFieldUnEmptyCount"] = len(entries) * 9
	return payload, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func detailIDs(n int) string {
	if n == 0 {
		return ""
	}
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprint(i)
	}
	return strings.Join(ids, ",") + ","
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func PayloadSummary(payload map[string]any) map[string]any {
	keys := []string{"src", "requestname", "field6433", "field6434", "field6437", "field8800", "nodesnum0", "submitdtlid0", "workflowid", "requestid"}
	summary := map[string]any{}
	for _, key := range keys {
		summary[key] = payload[key]
	}
	n, _ := parseFloat(payload["nodesnum0"])
	rows := make([]map[string]any, 0, int(n))
	for i := 0; i < int(n); i++ {
		rows = append(rows, map[string]any{
			"date":         payload[fmt.Sprintf("field6460_%d", i)],
			"project_id":   payload[fmt.Sprintf("field6448_%d", i)],
			"project_name": payload[fmt.Sprintf("field6448_%dname", i)],
			"hours":        payload[fmt.Sprintf("field6463_%d", i)],
			"progress":     payload[fmt.Sprintf("field6464_%d", i)],
			"content":      payload[fmt.Sprintf("field6449_%d", i)],
		})
	}
	summary["rows"] = rows
	return summary
}
