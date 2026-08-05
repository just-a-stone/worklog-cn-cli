package worklog

import "time"

const (
	WorkflowID   = 46
	WorkflowType = 22
	FormID       = -22
	NodeID       = 184
	ModeID       = 23
)

var WeekdayCN = []string{"星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日"}

type Project struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Code        string         `json:"code"`
	ManagerID   string         `json:"manager_id"`
	ManagerName string         `json:"manager_name"`
	Raw         map[string]any `json:"-"`
}

type HistoryRecord struct {
	RequestID      string         `json:"requestid"`
	Title          string         `json:"title"`
	StartDate      string         `json:"start_date,omitempty"`
	EndDate        string         `json:"end_date,omitempty"`
	ImportantLevel string         `json:"important_level,omitempty"`
	CreateTime     string         `json:"create_time,omitempty"`
	Link           string         `json:"link,omitempty"`
	UserID         string         `json:"userid,omitempty"`
	Raw            map[string]any `json:"-"`
}

type RequestDetailRow struct {
	WorkDate    string         `json:"date"`
	Weekday     string         `json:"weekday,omitempty"`
	Hours       float64        `json:"hours"`
	Content     string         `json:"content,omitempty"`
	Progress    float64        `json:"progress"`
	ProjectID   string         `json:"project_id,omitempty"`
	ProjectName string         `json:"project_name,omitempty"`
	ProjectCode string         `json:"project_code,omitempty"`
	ManagerID   string         `json:"manager_id,omitempty"`
	ManagerName string         `json:"manager_name,omitempty"`
	AutoCreated string         `json:"auto_created,omitempty"`
	KeyID       string         `json:"keyid,omitempty"`
	Raw         map[string]any `json:"-"`
}

type RequestDetail struct {
	RequestID      string             `json:"requestid"`
	Title          string             `json:"title,omitempty"`
	StartDate      string             `json:"start_date,omitempty"`
	EndDate        string             `json:"end_date,omitempty"`
	SubmitDate     string             `json:"submit_date,omitempty"`
	DurationDays   string             `json:"duration_days,omitempty"`
	LastTSEnd      string             `json:"last_ts_end,omitempty"`
	SubmitterID    string             `json:"submitter_id,omitempty"`
	SubmitterName  string             `json:"submitter_name,omitempty"`
	DepartmentID   string             `json:"department_id,omitempty"`
	DepartmentName string             `json:"department_name,omitempty"`
	ApproverID     string             `json:"approver_id,omitempty"`
	ApproverName   string             `json:"approver_name,omitempty"`
	ApprovalStatus string             `json:"approval_status,omitempty"`
	FeishuInstance string             `json:"feishu_instance,omitempty"`
	NextWeekPlan   string             `json:"next_week_plan,omitempty"`
	RequestLevel   string             `json:"request_level,omitempty"`
	NodeID         int                `json:"nodeid,omitempty"`
	WorkflowID     int                `json:"workflowid,omitempty"`
	Entries        []RequestDetailRow `json:"entries"`
	WeeklySummary  []map[string]any   `json:"weekly_summary,omitempty"`
	RawMain        map[string]any     `json:"-"`
	RawDetail      map[string]any     `json:"-"`
	RawParams      map[string]any     `json:"-"`
}

type WorkEntry struct {
	WorkDate    time.Time `json:"date"`
	Hours       float64   `json:"hours"`
	Content     string    `json:"content"`
	Progress    float64   `json:"progress"`
	ProjectID   string    `json:"project_id,omitempty"`
	ProjectName string    `json:"project_name,omitempty"`
	ProjectCode string    `json:"project_code,omitempty"`
	ManagerID   string    `json:"manager_id,omitempty"`
}

type FormContext struct {
	UserID                 string
	Username               string
	DeptID                 string
	DeptName               string
	WorkflowID             int
	NodeID                 int
	FormID                 int
	IsBill                 string
	WorkflowType           string
	LinkageUUID            string
	SignatureSecretKey     string
	SignatureAttributesStr string
	SubmitTokenName        string
	SubmitToken            string
	RequestNameDefault     string
	RawParams              map[string]any `json:"-"`
	RawSubmit              map[string]any `json:"-"`
}

type AccountInfo struct {
	UserID   string `json:"userid"`
	Username string `json:"username"`
	DeptID   string `json:"deptid"`
	DeptName string `json:"deptname"`
}

type PreparedSubmit struct {
	Context        map[string]any   `json:"context"`
	Entries        []map[string]any `json:"entries"`
	Payload        map[string]any   `json:"payload"`
	PayloadSummary map[string]any   `json:"payload_summary"`
	Form           FormContext      `json:"-"`
}
