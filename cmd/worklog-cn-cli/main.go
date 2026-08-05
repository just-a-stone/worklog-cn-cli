package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"worklog-cn-cli/internal/worklog"
)

const (
	exitOK      = 0
	exitError   = 1
	exitUsage   = 2
	exitRemote  = 3
	exitRefused = 4
)

type cliOptions struct {
	base          string
	cookie        string
	envFile       string
	profile       string
	timeout       float64
	insecure      bool
	format        string
	jsonOutput    bool
	fields        string
	yes           bool
	readonly      bool
	passwordStdin bool
}

func main() { os.Exit(run(os.Args[1:])) }

func run(argv []string) int {
	if option, ok := forbiddenPasswordFlag(argv); ok {
		fmt.Fprintf(os.Stderr, "ERROR: %s 不接受明文密码参数；请使用 ECOLOGY_PASSWORD 或 --password-stdin。\n", option)
		return exitUsage
	}
	options := &cliOptions{format: "json"}
	root := buildRoot(options)
	root.SetArgs(argv)
	if err := root.Execute(); err != nil {
		var typed *worklog.Error
		if errors.As(err, &typed) {
			switch typed.Kind {
			case worklog.KindUsage, worklog.KindConfig:
				fmt.Fprintf(os.Stderr, "ERROR: %s\n", typed)
				if typed.Kind == worklog.KindUsage {
					return exitUsage
				}
				return exitError
			case worklog.KindRefused:
				fmt.Fprintf(os.Stderr, "拒绝: %s\n", typed)
				return exitRefused
			case worklog.KindRemote:
				fmt.Fprintf(os.Stderr, "服务端错误: %s\n", typed)
				return exitRemote
			default:
				fmt.Fprintf(os.Stderr, "ERROR: %s\n", typed)
				return exitError
			}
		}
		if isUsageError(err) {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			return exitUsage
		}
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return exitError
	}
	return exitOK
}

func isUsageError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	for _, marker := range []string{"unknown command", "unknown flag", "unknown shorthand flag", "accepts no arg", "requires at least", "accepts at most", "flag needs an argument"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func buildRoot(options *cliOptions) *cobra.Command {
	root := &cobra.Command{Use: "worklog-cn-cli", Short: "Ecology Timesheet Go CLI", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().StringVar(&options.base, "base", "", "Ecology 服务地址")
	root.PersistentFlags().StringVar(&options.cookie, "cookie", "", "JSESSIONID 或完整 Cookie")
	root.PersistentFlags().StringVar(&options.envFile, "env-file", "", "显式 .env 文件路径")
	root.PersistentFlags().StringVar(&options.profile, "profile", "", "读取 .env.<profile>")
	root.PersistentFlags().Float64Var(&options.timeout, "timeout", 0, "请求超时秒数")
	root.PersistentFlags().BoolVar(&options.insecure, "insecure", false, "跳过 HTTPS 证书校验")
	root.PersistentFlags().StringVar(&options.format, "format", "json", "输出格式：json、jsonl、table、csv")
	root.PersistentFlags().BoolVar(&options.jsonOutput, "json", false, "输出 JSON（兼容 Python 脚本参数）")
	root.PersistentFlags().StringVar(&options.fields, "fields", "", "table/csv 输出的逗号分隔字段")
	root.PersistentFlags().BoolVarP(&options.yes, "yes", "y", false, "确认执行写操作")
	root.PersistentFlags().BoolVar(&options.readonly, "readonly", false, "拒绝所有写操作")
	root.PersistentFlags().BoolVar(&options.passwordStdin, "password-stdin", false, "从 stdin 读取密码")
	root.AddCommand(newLoginCommand(options))
	root.AddCommand(newWhoamiCommand(options))
	root.AddCommand(newProjectsCommand(options))
	root.AddCommand(newHistoryCommand(options))
	root.AddCommand(newViewCommand(options))
	root.AddCommand(newDryRunCommand(options))
	root.AddCommand(newSubmitCommand(options))
	return root
}

func (o *cliOptions) config() (worklog.Config, error) {
	cfg, err := worklog.LoadConfig(worklog.ConfigOptions{EnvFile: o.envFile, Profile: o.profile})
	if err != nil {
		return worklog.Config{}, err
	}
	if o.base != "" {
		cfg.BaseURL = o.base
	}
	if o.cookie != "" {
		cfg.Cookie = o.cookie
	}
	if o.timeout > 0 {
		cfg.Timeout = time.Duration(o.timeout * float64(time.Second))
	}
	if o.insecure {
		cfg.VerifyTLS = false
	}
	return cfg, nil
}

func (o *cliOptions) client(requireCookie bool) (*worklog.Client, worklog.Config, error) {
	cfg, err := o.config()
	if err != nil {
		return nil, worklog.Config{}, err
	}
	if requireCookie {
		cookie, err := worklog.ResolveCookie(o.cookie, cfg)
		if err != nil {
			return nil, worklog.Config{}, err
		}
		cfg.Cookie = cookie
	}
	client, err := worklog.NewClient(cfg)
	return client, cfg, err
}

func (o *cliOptions) output(value any) error {
	format := o.outputFormat()
	fields := []string{}
	for _, field := range strings.Split(o.fields, ",") {
		if field = strings.TrimSpace(field); field != "" {
			fields = append(fields, field)
		}
	}
	return worklog.Render(value, worklog.OutputOptions{Format: format, Fields: fields, Writer: os.Stdout})
}

func (o *cliOptions) outputFormat() string {
	if o.jsonOutput {
		return "json"
	}
	return strings.ToLower(strings.TrimSpace(o.format))
}

func forbiddenPasswordFlag(argv []string) (string, bool) {
	for _, arg := range argv {
		if arg == "--password" || strings.HasPrefix(arg, "--password=") {
			return arg, true
		}
	}
	return "", false
}

func commandAliases(primary string) []string {
	return []string{primary + ".py"}
}
