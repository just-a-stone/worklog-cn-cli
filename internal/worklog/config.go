package worklog

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://ecology.example.invalid"

type ConfigOptions struct {
	EnvFile string
	Profile string
}

type Config struct {
	BaseURL        string
	Username       string
	Password       string
	Cookie         string
	DefaultProject string
	Timeout        time.Duration
	VerifyTLS      bool
	EnvFile        string
}

func LoadConfig(opts ConfigOptions) (Config, error) {
	values := map[string]string{}
	path := opts.EnvFile
	if path == "" {
		path = findEnvFile(opts.Profile)
	}
	if path != "" {
		loaded, err := parseDotenvFile(path)
		if err != nil {
			return Config{}, &Error{Kind: KindConfig, Msg: "读取配置文件失败", Err: err}
		}
		values = loaded
	}
	get := func(keys ...string) string {
		for _, key := range keys {
			if value := strings.TrimSpace(os.Getenv(key)); value != "" {
				return value
			}
		}
		for _, key := range keys {
			if value := strings.TrimSpace(values[key]); value != "" {
				return value
			}
		}
		return ""
	}

	cfg := Config{
		BaseURL:        get("ECOLOGY_BASE", "ECOLOGY_BASE_URL", "WORKLOG_BASE"),
		Username:       get("ECOLOGY_USERNAME", "WORKLOG_USER"),
		Password:       get("ECOLOGY_PASSWORD", "WORKLOG_PASS"),
		Cookie:         get("ECOLOGY_JSESSIONID", "ECOLOGY_COOKIE", "JSESSIONID"),
		DefaultProject: get("ECOLOGY_DEFAULT_PROJECT"),
		EnvFile:        path,
		Timeout:        30 * time.Second,
		VerifyTLS:      true,
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if raw := get("ECOLOGY_TIMEOUT"); raw != "" {
		seconds, err := strconv.ParseFloat(raw, 64)
		if err != nil || seconds <= 0 {
			return Config{}, configError("ECOLOGY_TIMEOUT 必须是正数")
		}
		cfg.Timeout = time.Duration(seconds * float64(time.Second))
	}
	if raw := get("ECOLOGY_VERIFY_TLS", "ECOLOGY_VERIFY_SSL"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, configError("ECOLOGY_VERIFY_TLS 必须是 true 或 false")
		}
		cfg.VerifyTLS = value
	}
	return cfg, nil
}

func findEnvFile(profile string) string {
	filename := ".env"
	if profile != "" {
		filename += "." + profile
	}
	if cwd, err := os.Getwd(); err == nil {
		path := cwd
		for i := 0; i <= 5; i++ {
			candidate := filepath.Join(path, filename)
			if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
				return candidate
			}
			parent := filepath.Dir(path)
			if parent == path {
				break
			}
			path = parent
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".config", "worklog-cn", filename)
		if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
			return candidate
		}
	}
	return ""
}

func parseDotenvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"')) {
			value = value[1 : len(value)-1]
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func ResolveCookie(cookie string, cfg Config) (string, error) {
	value := strings.TrimSpace(cookie)
	if value == "" {
		value = strings.TrimSpace(cfg.Cookie)
	}
	if value == "" {
		return "", configError("缺少会话 Cookie，请设置 ECOLOGY_JSESSIONID 或使用 --cookie")
	}
	return value, nil
}

func WriteEnvValues(path string, updates map[string]string) error {
	if path == "" {
		path = filepath.Join(currentDir(), ".env")
	}
	var lines []string
	if raw, err := os.ReadFile(path); err == nil {
		lines = strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	} else if !os.IsNotExist(err) {
		return err
	} else {
		lines = []string{
			"# 本地会话与凭据（勿提交）",
			"ECOLOGY_BASE=" + DefaultBaseURL,
			"ECOLOGY_JSESSIONID=",
			"ECOLOGY_USERNAME=",
			"ECOLOGY_PASSWORD=",
		}
	}
	seen := map[string]bool{}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		key, _, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.TrimPrefix(key, "export "))
		if value, ok := updates[key]; ok {
			lines[i] = key + "=" + value
			seen[key] = true
		}
	}
	for key, value := range updates {
		if !seen[key] {
			lines = append(lines, key+"="+value)
		}
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		return err
	}
	_ = os.Chmod(path, 0600)
	return nil
}

func currentDir() string {
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	return "."
}
