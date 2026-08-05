package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"worklog-cn-cli/internal/worklog"
)

func newLoginCommand(options *cliOptions) *cobra.Command {
	var username string
	var whoami bool
	var writeEnv bool
	cmd := &cobra.Command{
		Use:     "login",
		Aliases: commandAliases("login"),
		Short:   "RSA 登录并可选写回 .env",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if whoami {
				return runWhoami(options)
			}
			cfg, err := options.config()
			if err != nil {
				return err
			}
			password := cfg.Password
			if options.passwordStdin {
				password, err = readPassword(os.Stdin)
				if err != nil {
					return worklog.UsageError("读取密码失败: %v", err)
				}
			}
			if username == "" {
				username = cfg.Username
			}
			if username == "" || password == "" {
				return worklog.UsageError("需要用户名和密码：--username 或 ECOLOGY_USERNAME；密码使用 ECOLOGY_PASSWORD 或 --password-stdin")
			}
			cfg.Cookie = ""
			client, err := worklog.NewClient(cfg)
			if err != nil {
				return err
			}
			result, err := client.Login(context.Background(), username, password)
			if err != nil {
				return err
			}
			if writeEnv {
				jsid := client.JSessionID()
				if jsid == "" {
					return worklog.RemoteError("登录成功但未获取 JSESSIONID")
				}
				path := cfg.EnvFile
				if path == "" {
					path = ".env"
				}
				if err := worklog.WriteEnvValues(path, map[string]string{"ECOLOGY_BASE": cfg.BaseURL, "ECOLOGY_USERNAME": username, "ECOLOGY_JSESSIONID": jsid}); err != nil {
					return worklog.ConfigError("写入 .env 失败: %v", err)
				}
				result["env_file"] = path
			}
			if options.outputFormat() == "table" || options.outputFormat() == "csv" {
				result["username"] = mask(stringValue(result["username"]))
			}
			result["jsessionid"] = maskSecret(stringValue(result["jsessionid"]))
			return options.output(result)
		},
	}
	cmd.Flags().StringVarP(&username, "username", "u", "", "登录账号；默认 ECOLOGY_USERNAME")
	cmd.Flags().BoolVar(&whoami, "whoami", false, "仅验证当前 Cookie")
	cmd.Flags().BoolVar(&writeEnv, "write-env", false, "登录成功后写入 .env")
	return cmd
}

func newWhoamiCommand(options *cliOptions) *cobra.Command {
	return &cobra.Command{Use: "whoami", Short: "验证当前会话并显示账号", Aliases: []string{"account"}, Args: cobra.NoArgs, RunE: func(_ *cobra.Command, _ []string) error { return runWhoami(options) }}
}

func runWhoami(options *cliOptions) error {
	client, _, err := options.client(true)
	if err != nil {
		return err
	}
	account, err := client.GetAccount(context.Background())
	if err != nil {
		return err
	}
	return options.output(map[string]any{"userid": account.UserID, "username": mask(account.Username), "deptid": account.DeptID, "deptname": account.DeptName, "jsessionid": maskSecret(client.JSessionID())})
}

func readPassword(reader io.Reader) (string, error) {
	data, err := io.ReadAll(bufio.NewReader(reader))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func mask(value string) string {
	runes := []rune(value)
	if len(runes) <= 2 {
		return strings.Repeat("*", len(runes))
	}
	return string(runes[0]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1])
}

func maskSecret(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return "(unknown)"
	}
	if len(runes) <= 4 {
		return "****"
	}
	return string(runes[:2]) + "****" + string(runes[len(runes)-2:])
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
