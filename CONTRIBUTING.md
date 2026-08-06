# 贡献指南

感谢你愿意改进 worklog-cn-cli。

## 开发环境

需要 Go 1.22 或更高版本，不需要 CGO。

```bash
git clone https://github.com/just-a-stone/worklog-cn-cli.git
cd worklog-cn-cli
go build ./cmd/worklog-cn-cli
```

## 提交前自检

```bash
go vet ./...
go test -race ./...
gofmt -l .          # 输出应为空
```

测试通过注入 HTTP RoundTripper 构造响应，**不访问真实 Ecology 服务**。新增功能请沿用这种方式，不要在测试里发起真实网络请求。

## 一些约定

- **不要提交 `.env`、Cookie、真实账号或真实工时数据。** `.gitignore` 已覆盖常见路径，但请在 `git diff` 里再确认一次。
- **不要在示例、测试和文档里写入真实的 `ECOLOGY_BASE`。** 统一使用 `https://ecology.example.invalid`。
- 涉及写接口（`requestOperation`）的改动，必须保持默认 dry-run 语义：没有 `--i-confirm` / `--yes` 时不得发出写请求。
- 字段 ID、workflow/node/form 常量与具体 Ecology 环境绑定。如果你的环境字段不同，请在 issue 中说明环境差异，不要直接把本地常量提交上来。

## Pull Request

1. 从 `master` 开分支。
2. 一个 PR 只做一件事，附上动机和验证方式。
3. 如果改了命令行为，同步更新 README 的命令参考。
4. CI（`go vet` + `go test -race`）必须为绿。

## 报告问题

请使用 issue 模板。粘贴日志前请脱敏：去掉服务地址、账号、Cookie、`requestid` 和工作内容正文。
