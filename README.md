<h1 align="center">worklog-cn-cli</h1>

<p align="center">
  <b>一条命令填完一周的泛微 Ecology 工时 —— 单文件二进制，默认只预览不提交。</b>
</p>

<p align="center">
  <a href="https://github.com/just-a-stone/worklog-cn-cli/actions/workflows/ci.yml"><img src="https://github.com/just-a-stone/worklog-cn-cli/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/just-a-stone/worklog-cn-cli/releases/latest"><img src="https://img.shields.io/github/v/release/just-a-stone/worklog-cn-cli?sort=semver" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/just-a-stone/worklog-cn-cli" alt="License"></a>
  <img src="https://img.shields.io/github/go-mod/go-version/just-a-stone/worklog-cn-cli" alt="Go version">
</p>

---

## 为什么

在 Ecology「01.Timesheet 提报」里填一周工时，要点开表单、逐天加明细行、选项目、等三段字段联动、填内容和进度，来回二十多次点击。每周一次，每次十分钟。

```bash
# 之前：浏览器里点二十几次
# 现在：
worklog-cn-cli submit --week 2026-08-05 --project 246 --content '开发与联调' --i-confirm
```

登录、表单 token、`linkageUUID`、字段联动、周边界计算全部自动处理。**没有 `--i-confirm` 时永远只预览，不会写服务端。**

> [!WARNING]
> `submit --i-confirm` / `submit --yes` 会真实调用 Ecology 写接口。第一次使用请先跑 `dry-run` 并核对生成的日期、项目、工时。

## 安装

**下载预编译二进制**（推荐，无需任何运行时）——到 [Releases](https://github.com/just-a-stone/worklog-cn-cli/releases/latest) 选择对应文件：

| 设备 | 文件 |
|---|---|
| Apple Silicon Mac（M1/M2/M3/M4） | `worklog-cn-cli-macos-arm64` |
| Intel Mac | `worklog-cn-cli-macos-amd64` |
| Linux x86_64 | `worklog-cn-cli-linux-amd64` |
| Linux ARM64 | `worklog-cn-cli-linux-arm64` |
| Windows | `worklog-cn-cli-windows-amd64.exe` |
| Windows ARM | `worklog-cn-cli-windows-arm64.exe` |

```bash
chmod +x worklog-cn-cli-macos-arm64
./worklog-cn-cli-macos-arm64 --version
```

每个 Release 同时提供 `checksums.txt`，建议先校验：`shasum -a 256 worklog-cn-cli-macos-arm64`。

**用 Go 安装**（需要 Go 1.22+）：

```bash
go install github.com/just-a-stone/worklog-cn-cli/cmd/worklog-cn-cli@latest
```

<details>
<summary><b>从源码构建</b></summary>

```bash
git clone https://github.com/just-a-stone/worklog-cn-cli.git
cd worklog-cn-cli
go build -trimpath -ldflags='-s -w' -o worklog-cn-cli ./cmd/worklog-cn-cli
```

交叉编译（`CGO_ENABLED=0`，产物为单文件静态二进制）：

```bash
CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -trimpath -o dist/worklog-cn-cli-linux-amd64      ./cmd/worklog-cn-cli
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o dist/worklog-cn-cli-darwin-arm64     ./cmd/worklog-cn-cli
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -o dist/worklog-cn-cli-windows-amd64.exe ./cmd/worklog-cn-cli
```

</details>

## 60 秒上手

**1. 配置** —— 在二进制同目录建 `.env`（可从 [`.env.example`](.env.example) 复制）：

```dotenv
ECOLOGY_BASE=https://ecology.example.invalid
ECOLOGY_USERNAME=your-account
ECOLOGY_PASSWORD=your-password
ECOLOGY_DEFAULT_PROJECT=246
```

**2. 登录**，把会话写回 `.env`：

```console
$ worklog-cn-cli login --write-env
```

**3. 看看有哪些项目**：

```console
$ worklog-cn-cli projects --format table --fields id,name,code,manager_name
id   name          code         manager_name
---  ------------  -----------  ------------
246  内部平台建设  XM-2026-018  张三
312  客户 A 交付   XM-2026-031  李四
```

**4. 预览这周的工时**（不写服务端）：

```console
$ worklog-cn-cli dry-run --week 2026-08-05 --project 246 --content '开发与联调' --format table
content     date        hours  progress  project_id  project_name
----------  ----------  -----  --------  ----------  ------------
开发与联调  2026-08-03  8      1         246         内部平台建设
开发与联调  2026-08-04  8      1         246         内部平台建设
开发与联调  2026-08-05  8      1         246         内部平台建设
开发与联调  2026-08-06  8      1         246         内部平台建设
开发与联调  2026-08-07  8      1         246         内部平台建设
```

`--week` 填该周任意一天即可：主表日期自动扩展为周一至周日，明细默认只含工作日（要周末加 `--include-weekend`）。

**5. 核对无误后提交**：

```console
$ worklog-cn-cli submit --week 2026-08-05 --project 246 --content '开发与联调' --i-confirm
```

## 每天写不同内容

用 JSON 文件替代 `--content`，支持数组和 `{"entries": [...]}` 两种顶层格式（见 [`examples/entries.json`](examples/entries.json)）：

```json
{
  "entries": [
    { "date": "2026-08-03", "project_id": "246", "hours": 8, "content": "需求对齐与开发", "progress": 1 },
    { "date": "2026-08-04", "project_id": "312", "hours": 8, "content": "客户联调",       "progress": 0.6 }
  ]
}
```

```bash
worklog-cn-cli dry-run -f examples/entries.json
```

字段名兼容多种别名，从表格导出的数据通常不用改列名：

| 业务字段 | 可用名称 |
| --- | --- |
| 日期 | `date`、`work_date`、`rq` |
| 工时 | `hours`、`gs`、`工时` |
| 内容 | `content`、`gznr`、`工作内容` |
| 进度 | `progress`、`wcjd`、`完成进度` |
| 项目 | `project_id`、`xmmc` |

## 安全设计

这是个会往 OA 里写数据的工具，所以默认全部关着：

- **写操作需要显式门闩。** `submit` 缺少 `--i-confirm` / `--yes` 时与 `dry-run` 完全等价；`--readonly` 会在建立写请求前直接拒绝。
- **密码不接受命令行参数。** `--password` 会被显式拒绝，避免泄漏到进程列表和 shell history。只支持 `ECOLOGY_PASSWORD` 或 `--password-stdin`：
  ```bash
  printf '%s\n' "$ECOLOGY_PASSWORD" | worklog-cn-cli --password-stdin login
  ```
- **`.env` 含密码和 Cookie**，已在 `.gitignore` 中排除，不要提交到版本库。
- **保持 TLS 校验。** `--insecure` 仅供本地排查，别留在日常流程里。

详见 [SECURITY.md](SECURITY.md)。

> [!IMPORTANT]
> 本项目是**非官方**第三方客户端，与泛微（Weaver）及 Ecology 产品无任何关联。请仅在你有权访问的实例上使用，并遵守所在组织的规定。
>
> 表单字段 ID、workflow/node/form 常量与**具体 Ecology 环境绑定**。跨环境使用前请先用 `dry-run --dump-payload` 核对 payload。

## 命令参考

| 命令 | 作用 | 写服务端 |
| --- | --- | --- |
| `login` | RSA 登录；`--write-env` 保存会话 | 否（仅写本地 `.env`） |
| `whoami` | 验证会话并显示当前账号、部门 | 否 |
| `projects` | 列出可选项目 | 否 |
| `history` | 列出最近提报记录 | 否 |
| `view REQUEST_ID` | 查看主表、明细和周汇总 | 否 |
| `dry-run` | 组装并展示 payload | 否 |
| `submit` | 预览；带 `--i-confirm` 时提交 | 仅确认后 |

<details>
<summary><b>全局参数与退出码</b></summary>

```text
--base URL                 Ecology 服务地址
--cookie VALUE             JSESSIONID 或完整 Cookie
--env-file PATH            指定 dotenv 文件
--profile NAME             读取 .env.NAME
--timeout SECONDS          请求超时
--insecure                 跳过 HTTPS 证书校验
--format json|jsonl|table|csv
--fields a,b,c             table/csv 输出列
--json                     强制 JSON 输出
--readonly                 拒绝写操作
--yes, -y                  确认写操作
--password-stdin           从 stdin 读取密码
--version                  显示版本
```

所有命令默认输出 JSON，便于 `jq` 处理。列表命令支持 `--format table|csv` 和 `--fields` 选列。

退出码：`0` 成功 · `1` 配置或网络错误 · `2` 参数错误 · `3` 服务端业务错误 · `4` 安全策略拒绝。

</details>

<details>
<summary><b>配置项与优先级</b></summary>

优先级：**命令行参数 > 进程环境变量 > `.env` > 默认值**。

程序从当前目录向上查找最多五级父目录定位 `.env`，也支持 `--env-file` 指定路径、`--profile NAME` 读取 `.env.NAME`。

| 变量 | 说明 | 默认 |
| --- | --- | --- |
| `ECOLOGY_BASE` | 服务地址 | 必填 |
| `ECOLOGY_USERNAME` | 账号 | 必填 |
| `ECOLOGY_PASSWORD` | 密码 | 登录时必填 |
| `ECOLOGY_JSESSIONID` | 会话 Cookie，`login --write-env` 自动写入 | — |
| `ECOLOGY_DEFAULT_PROJECT` | 默认项目 id | — |
| `ECOLOGY_TIMEOUT` | 请求超时秒数 | `30` |
| `ECOLOGY_VERIFY_TLS` | 是否校验证书 | `true` |

</details>

## FAQ

<details>
<summary><b>提示 <code>getAccountList 失败</code> 或会话无效</b></summary>

`ECOLOGY_JSESSIONID` 过期了。重新执行 `worklog-cn-cli login --write-env` 刷新会话。
</details>

<details>
<summary><b>macOS 提示"无法验证开发者"</b></summary>

先用 `checksums.txt` 校验文件，确认来源后移除下载隔离标记：

```bash
shasum -a 256 worklog-cn-cli-macos-arm64   # 与 checksums.txt 比对
xattr -d com.apple.quarantine worklog-cn-cli-macos-arm64
```
</details>

<details>
<summary><b>提交后字段错位 / 项目名为空</b></summary>

字段 ID 与 Ecology 环境绑定。用 `dry-run --dump-payload /tmp/p.json` 导出完整 payload，和浏览器 DevTools 里真实提交的请求体对比。如果确实不同，请开 issue 说明环境差异。
</details>

<details>
<summary><b>联动查询很慢</b></summary>

加 `--no-linkage` 跳过项目编号和经理的联动请求。注意仍会读取表单和项目列表，且部分字段会留空。
</details>

## 开发

```bash
go vet ./...
go test -race ./...
```

测试通过注入 HTTP RoundTripper 构造响应来验证请求组装、RSA 加密、Cookie 处理、响应解析和 payload，**不访问真实 Ecology 服务**。

```text
cmd/worklog-cn-cli/        Cobra 命令、参数、安全门闩和退出码
internal/worklog/
  config.go                dotenv、配置优先级和会话回写
  transport.go             HTTP、Cookie、超时和响应解码
  client.go                Ecology API 与动态表单上下文
  entries.go               entries 解析、周边界和工作日生成
  payload.go               Timesheet payload 组装
  output.go                JSON/JSONL/table/CSV 渲染
examples/entries.json      JSON entries 示例
```

动态表单的 token、签名字段和 `linkageUUID` 每次从 `loadForm` 重新获取，不复用过期 payload。

欢迎贡献，请先读 [CONTRIBUTING.md](CONTRIBUTING.md)。

## License

[MIT](LICENSE)
