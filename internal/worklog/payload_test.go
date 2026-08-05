package worklog

import (
	"testing"
	"time"
)

func TestBuildSubmitPayloadSortsRowsAndFormatsFields(t *testing.T) {
	ctx := FormContext{UserID: "7", Username: "alice", DeptID: "9", WorkflowID: WorkflowID, NodeID: NodeID, FormID: FormID, IsBill: "1", WorkflowType: "22", LinkageUUID: "uuid", SignatureSecretKey: "secret", SignatureAttributesStr: "attrs", SubmitTokenName: "7_46_addrequest_submit_token", SubmitToken: "token"}
	entries := []WorkEntry{
		{WorkDate: time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC), Hours: 7.5, Content: "later", Progress: 0.5, ProjectID: "246", ProjectName: "项目", ProjectCode: "P-246", ManagerID: "8"},
		{WorkDate: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), Hours: 8, Content: "earlier", Progress: 1, ProjectID: "246", ProjectName: "项目", ProjectCode: "P-246", ManagerID: "8"},
	}
	start := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	payload, err := BuildSubmitPayload(ctx, entries, &start, &end, "2026-07-31", "标题", "submit", "下周计划", time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if payload["field6460_0"] != "2026-08-03" || payload["field6460_1"] != "2026-08-05" {
		t.Fatalf("rows were not sorted: %#v", payload)
	}
	if payload["field6463_1"] != "7.50" || payload["field6464_1"] != "0.50" {
		t.Fatalf("numeric formatting mismatch: %#v", payload)
	}
	if payload["field6461"] != 1 || payload["field6437"] != 7 || payload["submitdtlid0"] != "0,1," {
		t.Fatalf("main fields mismatch: %#v", payload)
	}
	if payload["detailFieldUnEmptyCount"] != 18 {
		t.Fatalf("detail count mismatch: %#v", payload["detailFieldUnEmptyCount"])
	}
}
