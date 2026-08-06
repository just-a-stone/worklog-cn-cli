# 安全说明

## 报告漏洞

请通过 GitHub 的 [Private vulnerability reporting](https://github.com/just-a-stone/worklog-cn-cli/security/advisories/new) 提交，不要开公开 issue。

报告请包含复现步骤和影响范围，**并去掉真实服务地址、账号、密码和 Cookie**。

## 本项目的凭据处理

- 密码**不接受**命令行参数。`--password` 会被显式拒绝，避免泄漏到进程列表和 shell history；只支持环境变量或 `--password-stdin`。
- `.env`（含密码与 `JSESSIONID`）默认被 `.gitignore` 排除，程序也不会把它写进任何输出。
- `--insecure` 会跳过 TLS 校验，仅供本地排查使用，不要在日常流程里保留。
- 写接口默认关闭。`submit` 在没有 `--i-confirm` / `--yes` 时与 `dry-run` 等价；`--readonly` 会在建立写请求前直接拒绝。

## 使用范围

本项目是**非官方**的第三方客户端，与泛微（Weaver）及 Ecology 产品无任何关联。请仅在你**有权访问**的 Ecology 实例上使用，并遵守所在组织的规定。
