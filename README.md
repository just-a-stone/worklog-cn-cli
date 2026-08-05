# worklog-cn-cli

Ecology「01.Timesheet 提报」的 Go CLI。它将原 Python 脚本迁移为一个可独立分发的二进制，覆盖登录、会话、项目查询、提报历史、工时单详情、按周 payload 生成和安全提交。

项目依赖 Go 运行时和少量 CLI 库，不需要 Python、pip 或虚拟环境。

> [!WARNING]
> `submit --i-confirm` 和 `submit --yes` 会调用 Ecology 写接口。真实提交前，请先执行 `dry-run` 并检查生成的日期、项目、工时和 payload。

## 功能

- RSA PKCS#1 v1.5 登录与 Cookie 会话管理
- `loadForm`、`detailData`、项目浏览器和三类字段联动接口
- 项目、历史记录和已提交工时单详情查询
- 按自然周生成工作日或含周末的 entries
- JSON 文件输入、字段别名兼容和 payload 导出
- JSON、JSONL、table、CSV 输出
- 默认 dry-run、`--readonly` 和确认参数安全门闩
- `CGO_ENABLED=0` 交叉编译为单文件二进制

## 开始使用

### 1. 构建

需要 Go 1.22 或更高版本：

```bash
go build -trimpath -ldflags='-s -w' -o worklog-cn-cli ./cmd/worklog-cn-cli
./worklog-cn-cli --help
```

交叉编译示例：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o dist/worklog-cn-cli-linux-amd64 ./cmd/worklog-cn-cli
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o dist/worklog-cn-cli-darwin-arm64 ./cmd/worklog-cn-cli
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -o dist/worklog-cn-cli-macos-amd64 ./cmd/worklog-cn-cli
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -o dist/worklog-cn-cli-windows-amd64.exe ./cmd/worklog-cn-cli
```

### 2. GitHub Release

推送符合 `v*.*.*` 格式的版本标签后，GitHub Actions 会自动运行测试并构建以下可执行文件：

- `worklog-cn-cli-macos-amd64`
- `worklog-cn-cli-macos-arm64`
- `worklog-cn-cli-windows-amd64.exe`
- `worklog-cn-cli-windows-arm64.exe`

四个文件和 `checksums.txt` 会上传到同一个 GitHub Release。发布示例：

```bash
git tag v1.0.0
git push origin v1.0.0
```

仓库需要允许 GitHub Actions 使用 `GITHUB_TOKEN` 写入 Releases；workflow 已声明 `contents: write` 权限。

### 3. 配置

复制 [`.env.example`](.env.example) 为 `.env`，填写账号和密码：

```dotenv
ECOLOGY_BASE=https://ecology.example.invalid
ECOLOGY_USERNAME=your-account
ECOLOGY_PASSWORD=your-password
ECOLOGY_JSESSIONID=
ECOLOGY_DEFAULT_PROJECT=246
ECOLOGY_TIMEOUT=30
ECOLOGY_VERIFY_TLS=true
```

配置优先级为：命令行参数 > 进程环境变量 > `.env` > 默认值。程序会从当前目录向上查找最多五级父目录，也支持显式的 `--env-file` 和 `--profile`。

密码不接受 `--password`，避免出现在进程列表和 shell history。可以使用环境变量，或通过 stdin 输入：

```bash
printf '%s\n' "$ECOLOGY_PASSWORD" | ./worklog-cn-cli --password-stdin login
```

登录成功后，将新会话写回 `.env`：

```bash
./worklog-cn-cli login --write-env
```

> [!NOTE]
> `ECOLOGY_JSESSIONID` 会过期。遇到 `getAccountList 失败` 或会话无效时，重新执行 `login --write-env`。`.env` 包含凭据和 Cookie，不要提交到版本库。

### 4. 只读查询

所有命令默认输出 JSON。列表命令可使用 `--format table` 或 `--format csv`，也可以用 `--fields` 选择列。

```bash
# 当前账号和部门
./worklog-cn-cli whoami

# 可选项目
./worklog-cn-cli projects --page-size 50 --format table

# 最近提报历史
./worklog-cn-cli history --limit 10 --format table

# 查看一张已提交工时单
./worklog-cn-cli view 97217 --format table
```

只读命令包括 `whoami`、`projects`、`history` 和 `view`。它们不会调用 `requestOperation`。

### 5. 生成 dry-run

按周生成时，`--week` 可以填写该周任意一天；主表日期自动扩展为周一至周日，明细默认只包含周一至周五：

```bash
./worklog-cn-cli dry-run \
  --week 2026-08-05 \
  --project 246 \
  --content '开发与联调' \
  --hours 8 \
  --progress 1.0 \
  --dump-payload /tmp/worklog-payload.json
```

需要包含周末时添加 `--include-weekend`。`--no-linkage` 会跳过项目编号和经理的联动查询，但仍会读取表单和项目列表。

### 6. 使用 JSON entries

支持 JSON 数组和 `{ "entries": [...] }` 两种顶层格式。示例见 [`examples/entries.json`](examples/entries.json)：

```json
{
  "entries": [
    {
      "date": "2026-08-03",
      "project_id": "246",
      "hours": 8,
      "content": "需求对齐与开发",
      "progress": 1
    }
  ]
}
```

支持的字段别名：

| 业务字段 | 可用名称 |
| --- | --- |
| 日期 | `date`、`work_date`、`rq` |
| 工时 | `hours`、`gs`、`工时` |
| 内容 | `content`、`gznr`、`工作内容` |
| 进度 | `progress`、`wcjd`、`完成进度` |
| 项目 | `project_id`、`xmmc` |

```bash
./worklog-cn-cli dry-run -f examples/entries.json
```

### 7. 提交

`submit` 在没有确认参数时与 `dry-run` 等价：

```bash
./worklog-cn-cli submit \
  --week 2026-08-05 \
  --project 246 \
  --content '开发与联调'
```

确认 payload 后，显式打开写入门闩：

```bash
./worklog-cn-cli submit \
  --week 2026-08-05 \
  --project 246 \
  --content '开发与联调' \
  --i-confirm
```

`--yes` 是 `--i-confirm` 的全局别名；`--readonly` 会在建立写请求前拒绝提交。

## 命令参考

| 命令 | 作用 | 写服务端 |
| --- | --- | --- |
| `login` | RSA 登录；`--write-env` 保存会话 | 否，除本地 `.env` 外 |
| `whoami` | 查询当前账号 | 否 |
| `projects` | 查询可选项目 | 否 |
| `history` | 查询最近提报记录 | 否 |
| `view REQUEST_ID` | 查询主表、明细和周汇总 | 否 |
| `dry-run` | 生成并展示 payload | 否 |
| `submit` | 预览或提交 payload | 仅确认后 |

旧 Python 脚本名称仍有别名：`list-projects`、`list-history`、`view-request`、`dry-run-report`、`submit-report`。

通用选项：

```text
--base URL                 Ecology 服务地址
--cookie VALUE             JSESSIONID 或完整 Cookie
--env-file PATH            指定 dotenv 文件
--profile NAME             读取 .env.NAME
--timeout SECONDS          请求超时
--insecure                 跳过 HTTPS 证书校验
--format json|jsonl|table|csv
--fields a,b,c             table/csv 输出列
--json                     兼容旧脚本，强制 JSON 输出
--readonly                 拒绝写操作
--yes                      确认写操作
```

退出码：`0` 成功，`1` 配置或网络错误，`2` 参数错误，`3` 服务端业务错误，`4` 安全策略拒绝。

## 项目结构

```text
cmd/worklog-cn-cli/        Cobra 命令、参数、安全门闩和退出码
internal/worklog/
  config.go                dotenv、配置优先级和会话回写
  transport.go             HTTP、Cookie、超时和响应解码
  client.go                Ecology API 与动态表单上下文
  entries.go               entries 解析、周边界和工作日生成
  payload.go               Timesheet payload 组装
  output.go                JSON/JSONL/table/CSV 渲染
docs/api-map.md            Ecology 接口字段和请求顺序记录
examples/entries.json      JSON entries 示例
```

动态表单的 token、签名字段和 `linkageUUID` 每次从 `loadForm` 获取，不复用旧 payload。当前字段 ID、workflow/node/form 常量与 Ecology 环境绑定，详细接口约定见 [`docs/api-map.md`](docs/api-map.md)。默认地址是 HTTP，生产环境建议使用 HTTPS 并保持 TLS 校验。

## 开发与验证

测试不访问真实 OA，而是注入 HTTP RoundTripper 验证请求、RSA 加密、Cookie、响应解析和 payload：

```bash
GOCACHE=/tmp/worklog-cn-cli-go-build go test ./...
GOCACHE=/tmp/worklog-cn-cli-go-build go test -race ./...
GOCACHE=/tmp/worklog-cn-cli-go-build go vet ./...
```

真实服务验证或提交前，建议遵循：登录刷新会话 → 执行 `dry-run` → 核对 payload → 必要时使用 `submit --i-confirm`。
