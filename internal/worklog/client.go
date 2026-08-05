package worklog

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (c *Client) GetRSAInfo(ctx context.Context) (map[string]string, error) {
	value, err := c.DoGet(ctx, "/rsa/weaver.rsa.GetRsaInfo", url.Values{"ts": []string{nowMillis()}})
	if err != nil {
		return nil, err
	}
	data := asMap(value)
	if data == nil || stringValue(data["rsa_pub"]) == "" {
		return nil, remoteError("GetRsaInfo 返回无效数据")
	}
	return map[string]string{
		"rsa_pub":  stringValue(data["rsa_pub"]),
		"rsa_code": stringValue(data["rsa_code"]),
		"rsa_flag": stringValueDefault(data["rsa_flag"], "``RSA``"),
	}, nil
}

func EncryptLoginField(plaintext, rsaPubBase64, rsaFlag, rsaCode string) (string, error) {
	if rsaFlag == "" {
		rsaFlag = "``RSA``"
	}
	der, err := base64.StdEncoding.DecodeString(rsaPubBase64)
	if err != nil {
		return "", fmt.Errorf("RSA 公钥 Base64 无效: %w", err)
	}
	var publicKey *rsa.PublicKey
	if parsed, err := x509.ParsePKIXPublicKey(der); err == nil {
		publicKey, _ = parsed.(*rsa.PublicKey)
	}
	if publicKey == nil {
		if parsed, err := x509.ParsePKCS1PublicKey(der); err == nil {
			publicKey = parsed
		}
	}
	if publicKey == nil {
		return "", fmt.Errorf("无法解析 RSA 公钥")
	}
	data, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, []byte(plaintext+rsaCode))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data) + rsaFlag, nil
}

func (c *Client) Login(ctx context.Context, username, password string) (map[string]any, error) {
	if strings.TrimSpace(username) == "" || password == "" {
		return nil, usageError("登录用户名和密码不能为空")
	}
	rsaInfo, err := c.GetRSAInfo(ctx)
	if err != nil {
		return nil, err
	}
	encUser, err := EncryptLoginField(username, rsaInfo["rsa_pub"], rsaInfo["rsa_flag"], rsaInfo["rsa_code"])
	if err != nil {
		return nil, remoteError("RSA 用户名加密失败: %v", err)
	}
	encPassword, err := EncryptLoginField(password, rsaInfo["rsa_pub"], rsaInfo["rsa_flag"], rsaInfo["rsa_code"])
	if err != nil {
		return nil, remoteError("RSA 密码加密失败: %v", err)
	}
	loginForm := url.Values{
		"islanguid":          {"7"},
		"loginid":            {encUser},
		"userpassword":       {encPassword},
		"dynamicPassword":    {""},
		"tokenAuthKey":       {""},
		"validatecode":       {""},
		"validateCodeKey":    {""},
		"logintype":          {"1"},
		"messages":           {""},
		"isie":               {"false"},
		"appid":              {""},
		"service":            {""},
		"isRememberPassword": {"true"},
	}
	data, err := c.DoForm(ctx, "POST", "/api/hrm/login/checkLogin", loginForm)
	if err != nil {
		return nil, err
	}
	result := asMap(data)
	if result == nil || !strings.EqualFold(stringValue(result["loginstatus"]), "true") {
		return nil, remoteError("登录失败: %s", formatLoginError(result))
	}
	_, _ = c.DoForm(ctx, "POST", "/api/hrm/login/remindLogin", url.Values{"logintype": {"1"}})
	account, err := c.GetAccount(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"userid":     account.UserID,
		"username":   account.Username,
		"deptid":     account.DeptID,
		"deptname":   account.DeptName,
		"jsessionid": c.JSessionID(),
		"raw":        result,
	}, nil
}

func formatLoginError(result map[string]any) string {
	if result == nil {
		return "empty response"
	}
	parts := []string{}
	for _, key := range []string{"msg", "msgcode", "status", "loginstatus"} {
		if value := stringValue(result[key]); value != "" {
			if key == "msg" {
				parts = append(parts, value)
			} else {
				parts = append(parts, key+"="+value)
			}
		}
	}
	if len(parts) == 0 {
		return "unknown response"
	}
	return strings.Join(parts, "; ")
}

func (c *Client) GetAccount(ctx context.Context) (AccountInfo, error) {
	value, err := c.DoGet(ctx, "/api/hrm/login/getAccountList", url.Values{"__random__": {nowMillis()}})
	if err != nil {
		return AccountInfo{}, err
	}
	data := asMap(value)
	if data == nil || stringValue(data["status"]) != "1" {
		return AccountInfo{}, remoteError("getAccountList 失败")
	}
	account := asMap(data["data"])
	return AccountInfo{UserID: stringValue(account["userid"]), Username: stringValue(account["username"]), DeptID: stringValue(account["deptid"]), DeptName: stringValue(account["deptname"])}, nil
}

func (c *Client) LoadForm(ctx context.Context, workflowID int) (FormContext, error) {
	account, err := c.GetAccount(ctx)
	if err != nil {
		return FormContext{}, err
	}
	ts := nowMillis()
	data, err := c.DoForm(ctx, "POST", "/api/workflow/reqform/loadForm", formValues(map[string]any{
		"beagenter": 0, "f_weaver_belongto_userid": "", "f_weaver_belongto_usertype": 0,
		"isagent": 0, "iscreate": 1, "menuIds": "1,12", "menuPathIds": "1,12",
		"preloadkey": ts, "timestamp": ts, "workflowid": workflowID,
	}))
	if err != nil {
		return FormContext{}, err
	}
	return parseFormContext(data, account, workflowID)
}

func parseFormContext(value any, account AccountInfo, workflowID int) (FormContext, error) {
	data := asMap(value)
	params := asMap(data["params"])
	submit := asMap(data["submitParams"])
	mainData := asMap(data["maindata"])
	if params == nil || submit == nil {
		return FormContext{}, remoteError("loadForm 返回缺少 params/submitParams")
	}
	userID := stringValueDefault(params["f_weaver_belongto_userid"], account.UserID)
	tokenName := userID + "_" + strconv.Itoa(workflowID) + "_addrequest_submit_token"
	token, ok := submit[tokenName]
	if !ok {
		for key, item := range submit {
			if strings.HasSuffix(key, "_addrequest_submit_token") {
				tokenName, token = key, item
				ok = true
				break
			}
		}
	}
	if !ok {
		return FormContext{}, remoteError("loadForm 缺少 submit token")
	}
	deptCell := asMap(mainData["field6431"])
	deptID := stringValueDefault(deptCell["value"], account.DeptID)
	deptName := account.DeptName
	if special := asArray(deptCell["specialobj"]); len(special) > 0 {
		deptName = stringValueDefault(asMap(special[0])["name"], deptName)
	}
	titleCell := asMap(mainData["field-1"])
	return FormContext{
		UserID: userID, Username: stringValueDefault(params["username"], account.Username), DeptID: deptID, DeptName: deptName,
		WorkflowID: intValueDefault(params["workflowid"], workflowID), NodeID: intValueDefault(params["nodeid"], NodeID), FormID: intValueDefault(params["formid"], FormID),
		IsBill: stringValueDefault(params["isbill"], "1"), WorkflowType: stringValueDefault(submit["workflowtype"], stringValueDefault(params["workflowtype"], fmt.Sprint(WorkflowType))),
		LinkageUUID: stringValueDefault(params["linkageUUID"], stringValue(submit["linkageUUID"])), SignatureSecretKey: stringValue(params["signatureSecretKey"]), SignatureAttributesStr: stringValue(params["signatureAttributesStr"]),
		SubmitTokenName: tokenName, SubmitToken: stringValue(token), RequestNameDefault: stringValueDefault(titleCell["value"], stringValue(params["titlename"])), RawParams: params, RawSubmit: submit,
	}, nil
}

func (c *Client) LoadDetail(ctx context.Context, form FormContext) (map[string]any, error) {
	ts := nowMillis()
	params := form.RawParams
	reqParams := map[string]any{
		"requestid": "-1", "workflowid": form.WorkflowID, "nodeid": form.NodeID, "formid": form.FormID, "isbill": form.IsBill,
		"f_weaver_belongto_userid": form.UserID, "f_weaver_belongto_usertype": "0", "signatureSecretKey": form.SignatureSecretKey,
		"signatureAttributesStr": form.SignatureAttributesStr, "nodetype": intValueDefault(params["nodetype"], 0), "iscreate": "1", "isviewonly": "0",
		"ismode": intValueDefault(params["ismode"], 2), "modeid": intValueDefault(params["modeid"], ModeID), "isagent": intValueDefault(params["isagent"], 0),
		"beagenter": intValueDefault(params["beagenter"], 0), "creater": intValueDefault(params["creater"], intValueDefaultString(form.UserID, 0)),
		"needconfirm": stringValueDefault(params["needconfirm"], "0"), "creatertype": intValueDefault(params["creatertype"], 0), "requestType": intValueDefault(params["requestType"], 2),
		"isSelfAuth": intValueDefault(params["isSelfAuth"], 1), "selectNextFlow": stringValueDefault(params["selectNextFlow"], "0"), "layouttype": intValueDefault(params["layouttype"], 0),
		"apiResultCacheKey": stringValueDefault(params["apiResultCacheKey"], ts),
	}
	value, err := c.DoForm(ctx, "POST", "/api/workflow/reqform/detailData", formValues(map[string]any{
		"beagenter": 0, "f_weaver_belongto_userid": "", "f_weaver_belongto_usertype": 0, "isagent": 0, "iscreate": 1,
		"menuIds": "1,12", "menuPathIds": "1,12", "preloadkey": ts, "timestamp": ts, "workflowid": form.WorkflowID,
		"detailmark": "detail_1,detail_2", "reqParams": jsonString(reqParams), "wfTestStr": "",
	}))
	if err != nil {
		return nil, err
	}
	return asMap(value), nil
}

func (c *Client) ListProjects(ctx context.Context, form FormContext, pageSize int) ([]Project, error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	value, err := c.DoGet(ctx, "/api/public/browser/data/161", formValues(map[string]any{
		"pageSize": pageSize, "current": 1, "min": 1, "max": pageSize, "companyId": 1, "type": "browser.xmda_llk", "fielddbtype": "browser.xmda_llk",
		"currenttime": nowMillis(), "nodataloading": 0, "requestid": -1, "workflowid": form.WorkflowID, "wfid": form.WorkflowID, "billid": form.FormID, "isbill": form.IsBill,
		"f_weaver_belongto_userid": form.UserID, "f_weaver_belongto_usertype": 0, "wf_isagent": 0, "wf_beagenter": 0, "wfTestStr": "", "fieldid": 6448,
		"viewtype": 1, "fromModule": "workflow", "wfCreater": form.UserID, "disabledConditionCache": "true", "__random__": nowMillis(),
	}))
	if err != nil {
		return nil, err
	}
	data := asMap(value)
	rows := asArray(data["datas"])
	if len(rows) == 0 {
		rows = asArray(data["data"])
	}
	projects := make([]Project, 0, len(rows))
	for _, item := range rows {
		row := asMap(item)
		if row == nil {
			continue
		}
		managerID, managerName := parseManagerHTML(stringValue(row["xmjl"]))
		projects = append(projects, Project{ID: stringValueDefault(row["id"], stringValue(row["randomFieldId"])), Name: stringValueDefault(row["xmmc"], stringValue(row["name"])), Code: stringValue(row["xmbh"]), ManagerID: managerID, ManagerName: managerName, Raw: row})
	}
	return projects, nil
}

func parseManagerHTML(text string) (string, string) {
	if text == "" {
		return "", ""
	}
	re := regexp.MustCompile(`cardInfo/(\d+)[^>]*>\s*([^<]+)`)
	match := re.FindStringSubmatch(text)
	if len(match) == 3 {
		return match[1], strings.TrimSpace(match[2])
	}
	return "", strings.TrimSpace(text)
}

func (c *Client) ListHistory(ctx context.Context, limit int) ([]HistoryRecord, error) {
	if limit <= 0 {
		return []HistoryRecord{}, nil
	}
	value, err := c.DoForm(ctx, "POST", "/api/portal/element/workflowtab", formValues(map[string]any{
		"eid": 13, "hpid": 2, "subCompanyId": 1, "styleid": "1573611682069", "ebaseid": 8, "tabid": 2, "__random__": nowMillis(),
	}))
	if err != nil {
		return nil, err
	}
	data := asMap(value)
	rows := asArray(data["data"])
	result := make([]HistoryRecord, 0, minInt(limit, len(rows)))
	dateRE := regexp.MustCompile(`开始日期:(\d{4}-\d{2}-\d{2}).*?结束日期:(\d{4}-\d{2}-\d{2})`)
	for _, item := range rows {
		row := asMap(item)
		requestName := asMap(row["requestname"])
		titleHTML := stringValue(requestName["name"])
		start, end := "", ""
		if match := dateRE.FindStringSubmatch(titleHTML); len(match) == 3 {
			start, end = match[1], match[2]
		}
		result = append(result, HistoryRecord{RequestID: stringValue(requestName["requestid"]), Title: stripHTML(titleHTML), StartDate: start, EndDate: end, ImportantLevel: stringValue(row["importantleve"]), CreateTime: stringValue(row["createtime"]), Link: stringValue(requestName["link"]), UserID: stringValue(requestName["f_weaver_belongto_userid"]), Raw: row})
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (c *Client) GetRequestDetail(ctx context.Context, requestID string) (RequestDetail, error) {
	account, err := c.GetAccount(ctx)
	if err != nil {
		return RequestDetail{}, err
	}
	ts := nowMillis()
	formValue, err := c.DoForm(ctx, "POST", "/api/workflow/reqform/loadForm", formValues(map[string]any{
		"beagenter": 0, "f_weaver_belongto_userid": account.UserID, "f_weaver_belongto_usertype": 0, "isagent": 0, "iscreate": 0,
		"requestid": requestID, "menuIds": "1,12", "menuPathIds": "1,12", "preloadkey": ts, "timestamp": ts, "workflowid": WorkflowID,
	}))
	if err != nil {
		return RequestDetail{}, err
	}
	formData := asMap(formValue)
	params := asMap(formData["params"])
	if params == nil {
		return RequestDetail{}, remoteError("读取 request %s 的 loadForm 失败", requestID)
	}
	reqParams := map[string]any{
		"requestid": requestID, "workflowid": intValueDefault(params["workflowid"], WorkflowID), "nodeid": intValueDefault(params["nodeid"], NodeID), "formid": intValueDefault(params["formid"], FormID),
		"isbill": stringValueDefault(params["isbill"], "1"), "f_weaver_belongto_userid": stringValueDefault(params["f_weaver_belongto_userid"], account.UserID), "f_weaver_belongto_usertype": "0",
		"signatureSecretKey": stringValue(params["signatureSecretKey"]), "signatureAttributesStr": stringValue(params["signatureAttributesStr"]), "nodetype": intValueDefault(params["nodetype"], 0), "iscreate": "0", "isviewonly": stringValueDefault(params["isviewonly"], "1"),
		"ismode": intValueDefault(params["ismode"], 2), "modeid": intValueDefault(params["modeid"], ModeID), "isagent": 0, "beagenter": 0, "creater": intValueDefault(params["creater"], intValueDefaultString(account.UserID, 0)), "needconfirm": stringValueDefault(params["needconfirm"], "0"),
		"creatertype": intValueDefault(params["creatertype"], 0), "requestType": intValueDefault(params["requestType"], 2), "isSelfAuth": intValueDefault(params["isSelfAuth"], 1), "selectNextFlow": stringValueDefault(params["selectNextFlow"], "0"), "layouttype": intValueDefault(params["layouttype"], 0), "apiResultCacheKey": stringValueDefault(params["apiResultCacheKey"], ts),
	}
	detailValue, err := c.DoForm(ctx, "POST", "/api/workflow/reqform/detailData", formValues(map[string]any{
		"beagenter": 0, "f_weaver_belongto_userid": "", "f_weaver_belongto_usertype": 0, "isagent": 0, "iscreate": 0, "menuIds": "1,12", "menuPathIds": "1,12", "preloadkey": ts, "timestamp": ts, "workflowid": reqParams["workflowid"], "requestid": requestID, "detailmark": "detail_1,detail_2", "reqParams": jsonString(reqParams), "wfTestStr": "",
	}))
	if err != nil {
		return RequestDetail{}, err
	}
	detail := asMap(detailValue)
	mainData := asMap(formData["maindata"])
	detailOne := asMap(detail["detail_1"])
	rows := asMap(detailOne["rowDatas"])
	entries := make([]RequestDetailRow, 0, len(rows))
	for _, item := range rows {
		row := asMap(item)
		if row == nil {
			continue
		}
		projectName := cellName(row["field6448"])
		managerName := cellName(row["field6477"])
		hours, err := cellFloat(row["field6463"])
		if err != nil {
			return RequestDetail{}, remoteError("request %s 明细工时无效", requestID)
		}
		progress, err := cellFloat(row["field6464"])
		if err != nil {
			return RequestDetail{}, remoteError("request %s 明细进度无效", requestID)
		}
		entries = append(entries, RequestDetailRow{WorkDate: cellValue(row["field6460"]), Weekday: cellValue(row["field6459"]), Hours: hours, Content: cellValue(row["field6449"]), Progress: progress, ProjectID: cellValue(row["field6448"]), ProjectName: projectName, ProjectCode: cellValue(row["field7110"]), ManagerID: cellValue(row["field6477"]), ManagerName: managerName, AutoCreated: cellValue(row["field6467"]), KeyID: cellValue(row["keyid"]), Raw: row})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].WorkDate+entries[i].KeyID < entries[j].WorkDate+entries[j].KeyID
	})
	weekly := []map[string]any{}
	for _, item := range asMap(asMap(detail["detail_2"])["rowDatas"]) {
		row := asMap(item)
		flat := map[string]any{}
		for key, value := range row {
			flat[key] = cellValue(value)
		}
		weekly = append(weekly, flat)
	}
	return RequestDetail{RequestID: requestID, Title: cellValue(mainData["field-1"]), StartDate: cellValue(mainData["field6433"]), EndDate: cellValue(mainData["field6434"]), SubmitDate: cellValue(mainData["field6432"]), DurationDays: cellValue(mainData["field6437"]), LastTSEnd: cellValue(mainData["field8800"]), SubmitterID: cellValue(mainData["field6430"]), SubmitterName: cellName(mainData["field6430"]), DepartmentID: cellValue(mainData["field6431"]), DepartmentName: cellName(mainData["field6431"]), ApproverID: cellValue(mainData["field6541"]), ApproverName: cellName(mainData["field6541"]), ApprovalStatus: cellValue(mainData["field6436"]), FeishuInstance: cellValue(mainData["field6435"]), NextWeekPlan: cellValue(mainData["field6520"]), RequestLevel: cellValue(mainData["field-2"]), NodeID: intValueDefault(params["nodeid"], NodeID), WorkflowID: intValueDefault(params["workflowid"], WorkflowID), Entries: entries, WeeklySummary: weekly, RawMain: mainData, RawDetail: detail, RawParams: params}, nil
}

func (c *Client) LinkageLastTSEnd(ctx context.Context, form FormContext) (string, error) {
	value, err := c.DoForm(ctx, "POST", "/api/workflow/linkage/reqFieldSqlResult", formValues(map[string]any{
		"requestid": -1, "workflowid": form.WorkflowID, "nodeid": form.NodeID, "formid": form.FormID, "isbill": form.IsBill, "triSource": 2, "showAI": 0, "triFieldid_185": "", "rowIndexStr_185": -1, "triTableMark_185": "main", "field6430": form.UserID, "linkageid": 185, "linkageUUID": form.LinkageUUID, "wfTestStr": "", "f_weaver_belongto_userid": form.UserID, "f_weaver_belongto_usertype": 0,
	}))
	if err != nil {
		return "", err
	}
	assign := asMap(asMap(value)["assignInfo_185"])
	field := asMap(asMap(assign["changeValue"])["field8800"])
	return stringValue(field["value"]), nil
}

func (c *Client) LinkageProjectCode(ctx context.Context, form FormContext, projectID string, row int) (string, error) {
	value, err := c.DoForm(ctx, "POST", "/api/workflow/linkage/reqFieldSqlResult", formValues(map[string]any{
		"requestid": -1, "workflowid": form.WorkflowID, "nodeid": form.NodeID, "formid": form.FormID, "isbill": form.IsBill, "triSource": 1, "showAI": 0, "triFieldid_18": 6448, "rowIndexStr_18": row, "triTableMark_18": "detail_1", fmt.Sprintf("field6448_%d", row): projectID, "linkageid": 18, "linkageUUID": form.LinkageUUID, "wfTestStr": "", "f_weaver_belongto_userid": form.UserID, "f_weaver_belongto_usertype": 0,
	}))
	if err != nil {
		return "", err
	}
	field := asMap(asMap(asMap(value)["assignInfo_18"])["changeValue"])[fmt.Sprintf("field7110_%d", row)]
	return stringValue(asMap(field)["value"]), nil
}

func (c *Client) LinkageProjectManager(ctx context.Context, form FormContext, projectID string, row int) (string, string, error) {
	value, err := c.DoForm(ctx, "POST", "/api/workflow/linkage/reqDataInputResult", formValues(map[string]any{
		"requestid": -1, "workflowid": form.WorkflowID, "nodeid": form.NodeID, "formid": form.FormID, "isbill": form.IsBill, "triSource": 1, "showAI": 0, "triFieldid_10": 6448, "rowIndexStr_10": row, "triTableMark_10": "detail_1", fmt.Sprintf("field6448_%d", row): projectID, "linkageid": 10, "linkageUUID": form.LinkageUUID, "wfTestStr": "", "f_weaver_belongto_userid": form.UserID, "f_weaver_belongto_usertype": 0,
	}))
	if err != nil {
		return "", "", err
	}
	field := asMap(asMap(asMap(value)["assignInfo_10"])["changeValue"])[fmt.Sprintf("field6477_%d", row)]
	fieldMap := asMap(field)
	name := ""
	if special := asArray(fieldMap["specialobj"]); len(special) > 0 {
		name = stringValue(asMap(special[0])["name"])
	}
	return stringValue(fieldMap["value"]), name, nil
}

func (c *Client) EnrichEntries(ctx context.Context, form FormContext, entries []WorkEntry, defaultProject string, resolveLinkage bool) ([]WorkEntry, error) {
	projects, err := c.ListProjects(ctx, form, 50)
	if err != nil {
		return nil, err
	}
	byID := map[string]Project{}
	for _, project := range projects {
		byID[project.ID] = project
	}
	codeCache, managerCache := map[string]string{}, map[string]string{}
	result := make([]WorkEntry, 0, len(entries))
	for index, entry := range entries {
		projectID := strings.TrimSpace(entry.ProjectID)
		if projectID == "" {
			projectID = strings.TrimSpace(defaultProject)
		}
		if projectID == "" {
			return nil, usageError("entry %s 缺少 project_id", entry.WorkDate.Format("2006-01-02"))
		}
		project := byID[projectID]
		name, code, managerID := entry.ProjectName, entry.ProjectCode, entry.ManagerID
		if name == "" {
			name = project.Name
		}
		if code == "" {
			code = project.Code
		}
		if managerID == "" {
			managerID = project.ManagerID
		}
		if resolveLinkage {
			if code == "" {
				if _, ok := codeCache[projectID]; !ok {
					value, err := c.LinkageProjectCode(ctx, form, projectID, index)
					if err != nil {
						return nil, err
					}
					codeCache[projectID] = value
				}
				code = codeCache[projectID]
			}
			if managerID == "" {
				if _, ok := managerCache[projectID]; !ok {
					value, _, err := c.LinkageProjectManager(ctx, form, projectID, index)
					if err != nil {
						return nil, err
					}
					managerCache[projectID] = value
				}
				managerID = managerCache[projectID]
			}
		}
		if name == "" {
			name = projectID
		}
		entry.ProjectID, entry.ProjectName, entry.ProjectCode, entry.ManagerID = projectID, name, code, managerID
		result = append(result, entry)
	}
	return result, nil
}

func (c *Client) PrepareSubmit(ctx context.Context, entries []WorkEntry, defaultProject string, startDate, endDate *time.Time, requestName, src string, resolveLinkage bool, nextWeekPlan string, now time.Time) (PreparedSubmit, error) {
	form, err := c.LoadForm(ctx, WorkflowID)
	if err != nil {
		return PreparedSubmit{}, err
	}
	if _, err := c.LoadDetail(ctx, form); err != nil {
		return PreparedSubmit{}, err
	}
	enriched, err := c.EnrichEntries(ctx, form, entries, defaultProject, resolveLinkage)
	if err != nil {
		return PreparedSubmit{}, err
	}
	lastTSEnd, err := c.LinkageLastTSEnd(ctx, form)
	if err != nil {
		lastTSEnd = ""
	}
	payload, err := BuildSubmitPayload(form, enriched, startDate, endDate, lastTSEnd, requestName, src, nextWeekPlan, now)
	if err != nil {
		return PreparedSubmit{}, err
	}
	entryMaps := make([]map[string]any, 0, len(enriched))
	for _, entry := range enriched {
		entryMaps = append(entryMaps, map[string]any{"date": entry.WorkDate.Format("2006-01-02"), "project_id": entry.ProjectID, "project_name": entry.ProjectName, "project_code": entry.ProjectCode, "manager_id": entry.ManagerID, "hours": entry.Hours, "progress": entry.Progress, "content": entry.Content})
	}
	return PreparedSubmit{Context: map[string]any{"userid": form.UserID, "username": form.Username, "deptid": form.DeptID, "deptname": form.DeptName, "workflowid": form.WorkflowID, "nodeid": form.NodeID, "formid": form.FormID, "submit_token_name": form.SubmitTokenName, "requestname_default": form.RequestNameDefault, "linkage_uuid": form.LinkageUUID}, Entries: entryMaps, Payload: payload, PayloadSummary: PayloadSummary(payload), Form: form}, nil
}

func (c *Client) CheckSecondAuth(ctx context.Context, form FormContext, src string) error {
	_, err := c.DoForm(ctx, "POST", "/api/workflow/secondauth/getSecondAuthConfig", formValues(map[string]any{"workflowid": form.WorkflowID, "nodeid": form.NodeID, "requestid": -1, "src": src}))
	return err
}

func (c *Client) RequestOperation(ctx context.Context, payload map[string]any) (any, error) {
	return c.DoForm(ctx, "POST", "/api/workflow/reqform/requestOperation", formValues(payload))
}

func asMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func asArray(value any) []any {
	if typed, ok := value.([]any); ok {
		return typed
	}
	return nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func stringValueDefault(value any, fallback string) string {
	if text := strings.TrimSpace(stringValue(value)); text != "" {
		return text
	}
	return fallback
}

func intValueDefault(value any, fallback int) int {
	if value == nil {
		return fallback
	}
	if parsed, err := parseFloat(value); err == nil {
		return int(parsed)
	}
	return fallback
}

func intValueDefaultString(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func cellFloat(value any) (float64, error) {
	text := strings.TrimSpace(cellValue(value))
	if text == "" {
		return 0, nil
	}
	return strconv.ParseFloat(text, 64)
}

func jsonString(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func cellValue(value any) string {
	if object := asMap(value); len(object) > 0 {
		return stringValue(object["value"])
	}
	return stringValue(value)
}

func cellName(value any) string {
	object := asMap(value)
	special := asArray(object["specialobj"])
	if len(special) > 0 {
		return stringValue(asMap(special[0])["name"])
	}
	return ""
}

func stripHTML(value string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return strings.TrimSpace(re.ReplaceAllString(value, ""))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
