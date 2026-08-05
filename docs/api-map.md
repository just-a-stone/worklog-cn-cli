# 工时登记 API 地图

> 来源：泛微 Ecology 内部系统（地址通过 `ECOLOGY_BASE` 配置）
> 抓包：`docs/HAR/`
> - 登录：`login_Archive [26-07-22 18-39-55].har`
> - 工时创建/提交：`timesheet_Archive [26-07-29 17-58-54].har`
> - 门户查询（跟踪tab）：`portal-track_Archive [26-07-29 23-41-32].har`
> - 门户查询：`portal_Archive [26-07-29 18-00-03].har`

**结论：** 工时登记不是独立业务表 API，而是走流程  
**「01.Timesheet提报」**（`workflowid=46`）的创建 + 表单提交。

---

## 0. 通用约定

| 项 | 值 |
|---|---|
| Base URL | 由 `ECOLOGY_BASE` 配置 |
| Content-Type（写接口） | `application/x-www-form-urlencoded; charset=utf-8` |
| 常用头 | `X-Requested-With: XMLHttpRequest` |
| 会话 Cookie | `ecology_JSessionid` / `JSESSIONID` |
| 登录后 Cookie | `loginidweaver`、`loginuuids`、`languageidweaver`、`__randcode__` |

示例账号上下文（HAR 中）：

| 字段 | 值 |
|---|---|
| userid | `10001` |
| 姓名 | 示例用户 |
| 部门 id | `9001`（示例部门） |
| 分部 | `9002`（示例分部） |

---

## 1. 最小调用链（自动化够用）

```text
1. GET  /rsa/weaver.rsa.GetRsaInfo
2. POST /api/hrm/login/checkLogin
3. POST /api/workflow/reqform/loadForm          # workflowid=46, iscreate=1
4. POST /api/workflow/reqform/detailData
5. GET  /api/public/browser/data/161            # 查项目 id
6. POST /api/workflow/linkage/reqFieldSqlResult # 可选：上次 ts 结束日 / 项目编号
7. POST /api/workflow/linkage/reqDataInputResult # 可选：项目经理
8. POST /api/workflow/secondauth/getSecondAuthConfig
9. POST /api/workflow/reqform/requestOperation  # src=submit
```

可砍掉：门户菜单、消息中心、心跳、流程图、mail/integration、大量 `formula/getCurrentDateTime`。

---

## 2. 登录链

来源 HAR：`26-07-22 18-39-55`

### 2.1 获取 RSA 公钥

```http
GET /rsa/weaver.rsa.GetRsaInfo?ts={ms}
```

响应示例：

```json
{
  "rsa_flag": "``RSA``",
  "rsa_pub": "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A...",
  "rsa_code": "00f8436e"
}
```

说明：

- `rsa_pub` 为 PKCS#8/SPKI Base64 公钥
- 页面加密算法（`js/rsa/rsa.js`）：`Base64(RSA_PKCS1(明文 + rsa_code)) + rsa_flag`
- `rsa_code` 每次 `GetRsaInfo` 都会变，必须与当次公钥一起使用；**不能**只加密明文
- `rsa_flag` 默认 `` `RSA` ``，追加在 Base64 密文之后

### 2.2 登录

```http
POST /api/hrm/login/checkLogin
Content-Type: application/x-www-form-urlencoded; charset=utf-8
```

| 参数 | 说明 |
|---|---|
| `islanguid` | `7`（中文） |
| `loginid` | RSA(账号 + rsa_code) + `` `RSA` `` |
| `userpassword` | RSA(密码 + rsa_code) + `` `RSA` `` |
| `logintype` | `1` |
| `isie` | `false` |
| `isRememberPassword` | `true` / `false` |
| `dynamicPassword` | 空（无动态口令时） |
| `validatecode` / `validateCodeKey` | 验证码相关，当前可空 |
| `tokenAuthKey` / `appid` / `service` / `messages` | 可空 |

成功响应：

```json
{
  "msgcode": "0",
  "msg": "登录成功",
  "loginstatus": "true",
  "userid": 10001,
  "sumpasswordwrong": "0",
  "access_token": "",
  "cusData": []
}
```

Set-Cookie 示例：

- `loginidweaver=10001`
- `languageidweaver=7`
- `loginuuids=10001`

### 2.3 登录后附属（可选）

| Method | Path | 作用 |
|---|---|---|
| POST | `/api/hrm/login/remindLogin` | 登录提醒；`logintype=1` |
| POST | `/api/hrm/password/isWeakPassword` | 弱密码检查；`password` 为 RSA 密文 |
| GET | `/api/hrm/login/getAccountList?__random__=` | 当前账号/部门信息 |

`getAccountList` 关键字段：

```json
{
  "status": "1",
  "data": {
    "userid": "10001",
    "username": "示例用户",
    "deptid": 9001,
    "deptname": "示例部门",
    "subcompanyid": 9002,
    "subcompanyname": "示例分部",
    "userLanguage": "7"
  }
}
```

---

## 3. 打开 Timesheet 表单

### 3.1 前端入口

```text
/spa/workflow/static4form/index.html
  #/main/workflow/req
  ?iscreate=1
  &workflowid=46
  &isagent=0
  &beagenter=0
  &f_weaver_belongto_userid=
  &f_weaver_belongto_usertype=0
  &menuIds=1,12
  &menuPathIds=1,12
  &preloadkey={ts}
  &timestamp={ts}
```

### 3.2 固定流程上下文（当前环境）

| 字段 | 值 | 含义 |
|---|---|---|
| `workflowid` | `46` | 01.Timesheet提报 |
| `workflowname` | `01.Timesheet提报` | |
| `workflowtype` | `22` | 流程类型 |
| `nodeid` | `184` | 创建节点 |
| `nodetype` | `0` | 创建 |
| `nodename` | `创建` | |
| `formid` | `-22` | 表单 id |
| `isbill` | `1` | bill 表单 |
| `modeid` | `23` | 模板 |
| `ismode` | `2` | |
| `requestid` | `-1` | 新建 |
| `iscreate` | `1` | 创建态 |
| `creater` / `currentUserid` | `10001` | 当前用户 |
| `requestType` | `2` | |

### 3.3 加载表单（必调）

```http
POST /api/workflow/reqform/loadForm
```

请求要点：

| 参数 | 示例 |
|---|---|
| `workflowid` | `46` |
| `iscreate` | `1` |
| `isagent` | `0` |
| `beagenter` | `0` |
| `f_weaver_belongto_userid` | 空或当前用户 |
| `f_weaver_belongto_usertype` | `0` |
| `menuIds` | `1,12` |
| `menuPathIds` | `1,12` |
| `preloadkey` / `timestamp` | 同入口时间戳 |

响应关键块：

| 块 | 用途 |
|---|---|
| `params` | 流程上下文、签名、标题模板 |
| `submitParams` | 提交骨架 + token |
| `maindata` | 主表默认值 |
| `tableInfo` | 字段元数据（id/name/label） |
| `linkageCfg` | 联动配置 |
| `browserInfo` | 浏览按钮类型 |

**每次新建必须重新 loadForm**，取出：

| 字段 | 来源 | 用途 |
|---|---|---|
| `signatureSecretKey` | `params` | 提交签名 |
| `signatureAttributesStr` | `params` | 提交签名；解码：`beagenter=0\|configid=-1\|isagent=0\|` |
| `linkageUUID` | `params` / `submitParams` | 联动/提交关联 |
| `{userid}_{workflowid}_addrequest_submit_token` | `submitParams` | 防重提交 token，如 `10001_46_addrequest_submit_token` |

### 3.4 明细默认值（必调）

```http
POST /api/workflow/reqform/detailData
```

表单参数除 loadForm 同类字段外：

| 参数 | 说明 |
|---|---|
| `detailmark` | `detail_1,detail_2` |
| `reqParams` | JSON 字符串，见下 |

`reqParams` **必须**包含（缺扩展字段会 HTTP 500 空 body）：

```json
{
  "requestid": "-1",
  "workflowid": 46,
  "nodeid": 184,
  "formid": -22,
  "isbill": "1",
  "f_weaver_belongto_userid": "10001",
  "f_weaver_belongto_usertype": "0",
  "signatureSecretKey": "...",
  "signatureAttributesStr": "...",
  "nodetype": 0,
  "iscreate": "1",
  "isviewonly": "0",
  "ismode": 2,
  "modeid": 23,
  "isagent": 0,
  "beagenter": 0,
  "creater": 10001,
  "needconfirm": "0",
  "creatertype": 0,
  "requestType": 2,
  "isSelfAuth": 1,
  "selectNextFlow": "0",
  "layouttype": 0,
  "apiResultCacheKey": 1785319035873
}
```

其中 `apiResultCacheKey / needconfirm / creatertype / requestType / isSelfAuth / selectNextFlow / layouttype` 从当次 `loadForm.params` 透传。

响应：

```json
{
  "detail_1": {
    "indexnum": 0,
    "addRowDefValue": {
      "field6477": {"value": "10001"},
      "field6467": {"value": "1"},
      "field6460": {"value": ""}
    },
    "rowDatas": {}
  },
  "detail_2": {
    "indexnum": 0,
    "addRowDefValue": {},
    "rowDatas": {}
  }
}
```

### 3.5 附属接口

| 优先级 | Method | Path | 作用 |
|---|---|---|---|
| 辅助 | POST | `/api/workflow/reqform/rightMenu` | 右键菜单：提交 / 保存 |
| 辅助 | POST | `/api/workflow/reqform/signInput` | 签字意见配置 |
| 辅助 | GET | `/api/workflow/reqform/scripts` | 前端脚本（日期/进度校验） |
| 可跳 | POST | `/api/workflow/layout/getFlowChartInfo` | 流程图 |
| 可跳 | POST | `/api/workflow/reqform/getFormTab` | 自定义 Tab（当前空） |

`rightMenu` 关键按钮：

- 提交：`doSubmitBack()` / `BTN_SUBBACKNAME`
- 保存：`doSave_nNew()` / `BTN_WFSAVE`

---

## 4. 表单字段映射

### 4.1 主表 main

| fieldId | fieldname | 标签 | 类型 | 提交样例 | 备注 |
|---|---|---|---|---|---|
| -1 / `requestname` | - | 标题 | 文本 | `01.Timesheet提报-示例用户-2026-07-29` | loadForm 预填 |
| -2 / `requestlevel` | - | 紧急程度 | 枚举 | `0` | 0 正常 / 1 重要 / 2 紧急 |
| 6430 | `tjr` | 提交人 | 人员 | `10001` | |
| 6431 | `tjbm` | 提交部门 | 部门 | `9001` | |
| 6432 | `tjrq` | 提交日期 | 日期 | `2026-07-29` | |
| **6433** | **`ksrq`** | **开始日期** | 日期 | `2026-07-20` | 必填 |
| **6434** | **`jsrq`** | **结束日期** | 日期 | `2026-07-26` | 必填 |
| 6437 | `sct` | 时长（天） | 数字 | `7` | 可由日期推算 |
| 8800 | `sctsdjsrq` | 上次 ts 结束日期 | 文本 | `2026-07-19` | 联动算出 |
| 6461 | `ksrqxzdsfwxqy` | 开始是否周一 | 选择 | `0` | |
| 6462 | `ksrqsfyz` | 开始日期是否有值 | 选择 | `1` | |
| 6465 | `ksrqzjz` | 开始日期中间值 | 日期 | | 脚本用 |
| 6520 | `xzgzap` | 下周工作安排 | 多行文本 | | 可选 |
| 6436 | `spzt` | 审批状态 | 选择 | 空 | 提交时通常空 |
| 6435 | `fsspslm` | 飞书审批实例码 | 文本 | 空 | |
| 6541 | `dqspr` | 当前审批人 | 人员 | 空 | |

### 4.2 明细 detail_1（真正工时行，核心）

| fieldId | fieldname | 标签 | 类型 | 提交样例 | 备注 |
|---|---|---|---|---|---|
| **6448** | **`xmmc`** | **项目名称** | browser `xmda_llk` | `246` | 同时带 `field6448_Nname` |
| 7110 | `xmbh` | 项目编号 | 文本 | `20241218027` | 选项目后联动 |
| 6477 | `xmjl` | 项目经理 | 人员 | `26` | 选项目后联动 |
| **6449** | **`gznr`** | **工作内容** | 多行文本 | 文本 | 必填 |
| 6459 | `xq` | 星期 | 文本 | `星期一` | |
| **6460** | **`rq`** | **日期** | 日期 | `2026-07-20` | 必填 |
| **6463** | **`gs`** | **工时** | 小数 | `8.00` | 必填 |
| **6464** | **`wcjd`** | **完成进度** | 小数 | `1.00` | **0–1** |
| 6467 | `sfzdcj` | 是否自动创建 | 选择 | `0` | |

明细提交约定（HAR 实包）：

| 参数 | 样例 | 含义 |
|---|---|---|
| `nodesnum0` | `5` | 实际提交行数 |
| `indexnum0` | `10` | 明细 index 总数 |
| `submitdtlid0` | `5,6,7,8,9,` | 真正提交的行号 |
| `deldtlid0` | 空 | 删除行 |
| 字段名 | `field{id}_{rowIndex}` | 如 `field6463_5=8.00` |
| 项目显示名 | `field6448_{row}name` | 如 `示例项目四期` |

说明：

- UI 可能先预生成空行 `0..9`，再填后几行提交
- API 侧可简化：从 `_0` 起编，`submitdtlid0=0,1,...,N-1,`，`indexnum0=N`，`nodesnum0=N`

### 4.3 明细 detail_2（周汇总，可先忽略）

按项目汇总周一～周日：

| fieldId | fieldname | 标签 |
|---|---|---|
| 6468 | `xmmc` | 项目名称 |
| 7111 | `xmbm` | 项目编码 |
| 6469 | `zy` | 周一 |
| 6470 | `ze` | 周二 |
| 6471 | `zs` | 周三 |
| 6472 | `zs1` | 周四 |
| 6473 | `zw` | 周五 |
| 6474 | `zl` | 周六 |
| 6475 | `zr` | 周日 |

HAR 中本次提交：`nodesnum1=0`，`indexnum1=0`。

---

## 5. 查询 / 联动 API

### 5.1 项目浏览器（查可选项目）

```http
GET /api/public/browser/data/161
  ?pageSize=10
  &current=1
  &min=1
  &max=10
  &companyId=1
  &type=browser.xmda_llk
  &fielddbtype=browser.xmda_llk
  &requestid=-1
  &workflowid=46
  &wfid=46
  &billid=-22
  &isbill=1
  &f_weaver_belongto_userid=10001
  &f_weaver_belongto_usertype=0
  &fieldid=6448
  &viewtype=1
  &fromModule=workflow
  &wfCreater=10001
  &__random__=
```

列：

- `xmmc` 项目名称
- `xmbh` 项目编号(流程)
- `xmjl` 项目经理（HTML）
- `xmcy` 项目成员（HTML）

HAR 可见项目：

| id | 名称 | 编号 |
|---|---|---|
| `246` | 示例项目四期 | `20241218027` |
| `56` | 请假专用 | `20240812018` |
| `14` | 售前费用 | `20240812001` |

配套：

| Method | Path | 作用 |
|---|---|---|
| GET | `/api/public/browser/condition/161?...` | 查询条件（项目名称等） |
| GET | `/api/common/browser/tab/list?type=161` | tab 配置（当前空） |

### 5.2 字段联动

#### 上次 Timesheet 结束日

```http
POST /api/workflow/linkage/reqFieldSqlResult
```

| 参数 | 值 |
|---|---|
| `linkageid` | `185` |
| `triSource` | `2` |
| `triTableMark_185` | `main` |
| `rowIndexStr_185` | `-1` |
| `field6430` | `10001`（提交人） |
| `workflowid/nodeid/formid/isbill/requestid` | 同上 |

响应：

```json
{
  "assignInfo_185": {
    "changeValue": {
      "field8800": {"value": "2026-07-19"}
    }
  }
}
```

#### 选项目 → 项目编号

```http
POST /api/workflow/linkage/reqFieldSqlResult
```

| 参数 | 值 |
|---|---|
| `linkageid` | `18` |
| `triSource` | `1` |
| `triFieldid_18` | `6448` |
| `triTableMark_18` | `detail_1` |
| `rowIndexStr_18` | 行号，如 `0` |
| `field6448_{row}` | 项目 id，如 `246` |

响应：`field7110_{row} = 项目编号`

#### 选项目 → 项目经理

```http
POST /api/workflow/linkage/reqDataInputResult
```

| 参数 | 值 |
|---|---|
| `linkageid` | `10` |
| `triSource` | `1` |
| `triFieldid_10` | `6448` |
| `triTableMark_10` | `detail_1` |
| `rowIndexStr_10` | 行号 |
| `field6448_{row}` | 项目 id |

响应示例：

```json
{
  "assignInfo_10": {
    "changeValue": {
      "field6477_0": {
        "value": "26",
        "specialobj": [{"id": "26", "name": "示例经理"}]
      }
    }
  }
}
```

### 5.3 公式 / 日历辅助

| Method | Path | 作用 |
|---|---|---|
| POST | `/api/workflow/formula/assignValue` | 公式赋值（日期等） |
| POST | `/api/workflow/formula/getCurrentDateTime` | 服务器时间；`type=CurrDateTime` |
| GET | `/api/esb/oa/execute?eventkey=getE9NewKQHoliday` | 节假日 |
| GET | `/api/esb/oa/execute?eventkey=QueryLeaveHours&params={"userId":"10001"}` | 请假工时 |

### 5.4 历史提报查询（门户 · 跟踪 tab）

来源 HAR：`[26-07-29 23-41-32].har`（门户「我的流程 · 跟踪」）。

这不是独立“工时业务表”查询 API，而是门户流程元素列表。对 Timesheet 够用：能拿到 `requestid`、标题里的周区间、创建时间。

```http
POST /api/portal/element/workflowtab
Content-Type: application/x-www-form-urlencoded; charset=utf-8
X-Requested-With: XMLHttpRequest
```

| 参数 | 样例 | 说明 |
|---|---|---|
| `eid` | `13` | 门户元素 id（当前环境固定） |
| `hpid` | `2` | 门户页 id |
| `subCompanyId` | `9002` | 分部 |
| `styleid` | `1573611682069` | 元素样式 id |
| `ebaseid` | `8` | 流程元素类型 |
| `tabid` | `2` | **跟踪** tab（本 HAR 用的） |

`tabids` 响应为 `["1","5","3","2","4"]`，其中 `tabid=2` 对应 `tabTitle=跟踪`。其它 tab（待办/已办等）未完整抓包，需要时再补。

响应关键结构：

```json
{
  "tabids": ["1", "5", "3", "2", "4"],
  "data": [
    {
      "requestname": {
        "requestid": "97180",
        "name": "01.Timesheet提报-示例用户-2026-07-29<B>（开始日期:2026-07-20, 结束日期:2026-07-26）</B>",
        "link": "/workflow/request/ViewRequestForwardSPA.jsp?requestid=97180&isovertime=",
        "f_weaver_belongto_userid": "10001",
        "f_weaver_belongto_usertype": "0"
      },
      "importantleve": "正常",
      "createtime": "17:58:51"
    }
  ],
  "tabsetting": {
    "count": "72",
    "from": "workflow",
    "more": "{\"perpage\":\"10\",\"tabTitle\":\"跟踪\",\"viewType\":\"4\",\"fromhp\":\"1\",...}"
  },
  "dataColoums": [
    {"fieldName": "标题", "id": "12"},
    {"fieldName": "紧急程度", "id": "87"},
    {"fieldName": "创建时间", "id": "114"}
  ]
}
```

字段解析约定（客户端已实现）：

| 字段 | 来源 | 说明 |
|---|---|---|
| `requestid` | `requestname.requestid` | 流程实例 id |
| `title` | `requestname.name` 去 HTML | 完整标题 |
| `start_date` / `end_date` | 从标题正则抽取 | `开始日期:YYYY-MM-DD` / `结束日期:YYYY-MM-DD` |
| `create_time` | `createtime` | 多为当天时分秒；跨天时可能只有时间 |
| `link` | `requestname.link` | 查看页相对路径 |
| `count` | `tabsetting.count` | 跟踪列表总条数（HAR 样例 72） |

HAR 样例（10 条，均为 Timesheet）：

| requestid | 周区间 | createtime |
|---|---|---|
| 97180 | 2026-07-20 ~ 2026-07-26 | 17:58:51 |
| 97179 | 2026-07-13 ~ 2026-07-19 | 17:18:23 |
| 97178 | 2026-07-06 ~ 2026-07-12 | 17:12:48 |
| 94324 | 2026-06-01 ~ 2026-06-07 | 15:51:20 |
| … | … | … |

局限：

1. 当前请求**没有**显式 `workflowid=46` 过滤；HAR 返回恰好全是 Timesheet，其它流程混入时需客户端按标题前缀过滤。
2. 本包只抓到第一页（`perpage=10`）；翻页/按日期过滤未覆盖。

### 5.5 已提交工时单明细回读（实网验证）

使用 `workflowtab` 返回的 `requestid` 回读。已对 `requestid=97180` 实测：主表周区间 `2026-07-20 ~ 2026-07-26`，`detail_1` 返回 5 行、合计 `40.00` 小时。

调用顺序：

```text
POST /api/workflow/reqform/loadForm    # requestid={id}, iscreate=0
POST /api/workflow/reqform/detailData  # requestid={id}, detailmark=detail_1,detail_2
```

`loadForm` 的最小上下文：

```text
requestid={id}&workflowid=46&iscreate=0&isagent=0&beagenter=0
```

`detailData.reqParams` 需从同一次 `loadForm.params` 透传：`requestid`、`workflowid`、`nodeid`、`formid`、`isbill`、`signatureSecretKey`、`signatureAttributesStr`、`isviewonly`、`ismode`、`modeid`、`needconfirm`、`creatertype`、`requestType`、`isSelfAuth`、`selectNextFlow`、`layouttype`、`apiResultCacheKey`。缺少扩展上下文可能导致服务端 500。

主表字段：

| 字段 | 含义 |
|---|---|
| `field-1` | 标题 |
| `field6432` | 提交日期 |
| `field6433` / `field6434` | 开始 / 结束日期 |
| `field6437` | 时长（天） |
| `field6430` / `field6431` | 提交人 / 提交部门 |
| `field6436` | 审批状态 |
| `field6541` | 当前审批人 |
| `field8800` | 上次 TS 结束日期 |

`detail_1.rowDatas` 是实际工时行；核心字段为 `field6448`（项目）、`field6449`（工作内容）、`field6459`（星期）、`field6460`（日期）、`field6463`（工时）、`field6464`（完成进度）、`field7110`（项目编号）、`field6477`（项目经理）。浏览型字段名在 `specialobj[0].name`，原始 id 在 `value`。

当前 CLI：

```bash
python3 scripts/list_history.py
python3 scripts/view_request.py 97180
python3 scripts/view_request.py 97180 --json
```

`view_request.py` 只读，不会保存、提交或修改流程。

旧接口 `POST /api/portal/element/workflow`（无 tab）仍可能出现在门户默认卡片刷新，信息更少；历史列表以 `workflowtab` 为准。

---

## 6. 提交（核心写接口）

### 6.1 二次认证检查

```http
POST /api/workflow/secondauth/getSecondAuthConfig
```

| 参数 | 值 |
|---|---|
| `workflowid` | `46` |
| `nodeid` | `184` |
| `requestid` | `-1` |
| `src` | `submit` |

当前环境：`isEnableAuth=0`，可忽略。

### 6.2 正式提交

```http
POST /api/workflow/reqform/requestOperation
Content-Type: application/x-www-form-urlencoded; charset=utf-8
X-Requested-With: XMLHttpRequest
Cookie: ecology_JSessionid=...; JSESSIONID=...; loginidweaver=10001; ...
```

#### 流程控制字段

| 参数 | 样例 | 说明 |
|---|---|---|
| `src` | `submit` | 提交；保存草稿对应 save 路径 |
| `actiontype` | `requestOperation` | |
| `workflowid` | `46` | |
| `workflowtype` | `22` | |
| `nodeid` | `184` | |
| `nodetype` | `0` | |
| `formid` | `-22` | |
| `isbill` | `1` | |
| `iscreate` | `1` | |
| `requestid` | `-1` | 新建 |
| `requestname` | `01.Timesheet提报-示例用户-2026-07-29` | 标题 |
| `requestlevel` | `0` | 紧急程度 |
| `f_weaver_belongto_userid` | `10001` | |
| `f_weaver_belongto_usertype` | `0` | |
| `lastloginuserid` | `10001` | |
| `needwfback` | `1` | |
| `selectNextFlow` | `0` | |
| `openDataVerify` | `0` | |
| `10001_46_addrequest_submit_token` | loadForm 时间戳 | **每次新建刷新** |
| `linkageUUID` | loadForm | |
| `signatureSecretKey` | loadForm | |
| `signatureAttributesStr` | loadForm | |
| `remark` | 空 | 签字意见 |
| `isOdocRequest` | `0` | |
| `mainFieldUnEmptyCount` | `9` | 非空主字段计数（UI 统计） |
| `detailFieldUnEmptyCount` | `45` | 非空明细字段计数 |

#### 主表字段（样例）

```text
field6430=10001
field6431=9001
field6432=2026-07-29
field6433=2026-07-20
field6434=2026-07-26
field6437=7
field8800=2026-07-19
field6461=0
field6462=1
field6436=
field6435=
field6520=
field6541=
field6541groupnum=0
```

#### 明细字段（样例，5 行工作日）

```text
nodesnum0=5
indexnum0=10
submitdtlid0=5,6,7,8,9,
deldtlid0=
nodesnum1=0
indexnum1=0
submitdtlid1=
deldtlid1=

field6448_5=246
field6448_5name=示例项目四期
field6449_5=AP 堆存费重跑，测试并加入完整流程
field6459_5=星期一
field6460_5=2026-07-20
field6463_5=8.00
field6464_5=1.00
field6467_5=0
field6477_5=26
field7110_5=20241218027
# ... _6 ~ _9 同结构
```

#### 校验相关（HAR 原样，实现时可跟随 UI 或精简验证）

- `verifyRequiredRange`：必填字段列表
- `existChangeRange`：变更字段列表

### 6.3 响应注意

HAR 中 `requestOperation` 的 `status=0` 且无 response body，像是抓包被中断/页面跳转。  
**请求体完整，响应未捕获**。实现时以实际返回 JSON 为准，成功后一般可在「我的流程」看到新 `requestid`。

---

## 7. 业务脚本约束（来自 `/api/workflow/reqform/scripts`）

前端脚本暴露的字段名（便于对照）：

| 区域 | 字段名 | 含义 |
|---|---|---|
| main | `tjr` | 提交人 |
| main | `ksrq` / `jsrq` | 开始/结束日期 |
| main | `sctsdjsrq` | 上次 ts 结束日期 |
| main | `ksrqzjz` | 开始日期中间值 |
| detail_1 | `xmmc` / `xq` / `rq` / `gznr` / `gs` / `wcjd` / `sfzdcj` | 工时行 |
| detail_2 | `xmmc` / `zy`~`zr` | 周汇总 |

已知校验：

- `wcjd`（完成进度）必须在 **0–1**，否则清空并提示
- 开始/结束日期会结合节假日、上次 ts 结束日做周偏移（脚本逻辑较重；API 提交时建议直接给合法工作周）

---

## 8. 实现注意点

1. **密码 RSA**：`loginid` / `userpassword` = Base64(RSA_PKCS1(明文 + rsa_code)) + `` `RSA` ``；`rsa_pub`/`rsa_code`/`rsa_flag` 均来自当次 `GetRsaInfo`。
2. **每次新建**必须重新 `loadForm`，刷新：
   - `signatureSecretKey`
   - `signatureAttributesStr`
   - `linkageUUID`
   - `{uid}_{wfid}_addrequest_submit_token`
3. **明细 index** 不强制从 UI 的 `5..9` 复制；API 可自 `0` 起连续编号，只要 `nodesnum0` / `indexnum0` / `submitdtlid0` 自洽。
4. 选项目后：
   - 可调联动 18/10 拿 `xmbh` / `xmjl`
   - 或对已知项目本地写死（如 246 → 编号 `20241218027`，经理 `26`）
5. Cookie 全程携带；登录前后 `__randcode__` 可能变化，以服务端 Set-Cookie 为准。
6. 历史列表用 `POST /api/portal/element/workflowtab`（`tabid=2` 跟踪）；用其中 `requestid` 以 `loadForm(iscreate=0) → detailData` 回读主表和工时明细。

---

## 9. 接口优先级总表

| 级别 | Method | Path | 场景 |
|---|---|---|---|
| P0 | GET | `/rsa/weaver.rsa.GetRsaInfo` | 登录 |
| P0 | POST | `/api/hrm/login/checkLogin` | 登录 |
| P0 | POST | `/api/workflow/reqform/loadForm` | 新建表单 / 已提交单主表回读 |
| P0 | POST | `/api/workflow/reqform/detailData` | 新建默认值 / 已提交单明细回读 |
| P0 | GET | `/api/public/browser/data/161` | 查项目 |
| P0 | POST | `/api/workflow/reqform/requestOperation` | 提交/保存 |
| P1 | POST | `/api/workflow/linkage/reqFieldSqlResult` | 上次结束日、项目编号 |
| P1 | POST | `/api/workflow/linkage/reqDataInputResult` | 项目经理 |
| P1 | POST | `/api/workflow/secondauth/getSecondAuthConfig` | 提交前检查 |
| P1 | POST | `/api/portal/element/workflowtab` | 历史列表（跟踪 tab） |
| P2 | POST | `/api/hrm/login/remindLogin` | 登录附属 |
| P2 | GET | `/api/hrm/login/getAccountList` | 用户信息 |
| P2 | POST | `/api/workflow/reqform/rightMenu` | 菜单 |
| P2 | GET | `/api/workflow/reqform/scripts` | 校验脚本 |
| P2 | POST | `/api/portal/element/workflow` | 门户默认流程卡片（信息更少） |
| P3 | 其他 portal/msg/mail/heartbeat | | 可忽略 |

---

## 10. HAR 覆盖缺口

| 缺口 | 说明 | 建议 |
|---|---|---|
| `requestOperation` 成功响应 | HAR status=0，无 body | 实网提交一次补抓 |
| 保存草稿完整包 | 仅见菜单 `doSave_nNew()` | 需要时再抓 save |
| 流程列表翻页/按日期/按 workflowid 过滤 | 仅跟踪 tab 首页 10 条 | 抓「我的请求」完整列表或 more 翻页 |
| 编辑已有 requestid | 当前仅 `iscreate=1` | 抓 ViewRequest / 再提交 |
| 验证码/二次认证开启态 | 当前关闭 | 环境变更时再补 |

---

## 11. 下一步

1. 客户端已覆盖：`login → loadForm → build rows → submit`、`list_history`（workflowtab）、`get_request_detail`（已提交单明细）
2. 实网 dry-run / 提交一次，固化 `requestOperation` 成功/失败响应结构
3. 如需改单或历史翻页/按日期过滤，补抓对应 HAR 并回填本节
