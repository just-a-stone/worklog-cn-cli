package worklog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigEnvironmentOverridesDotenv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("ECOLOGY_BASE=http://dotenv\nECOLOGY_USERNAME='from-file'\nECOLOGY_TIMEOUT=4\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ECOLOGY_USERNAME", "from-env")
	t.Setenv("ECOLOGY_BASE", "")
	cfg, err := LoadConfig(ConfigOptions{EnvFile: path})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Username != "from-env" || cfg.BaseURL != "http://dotenv" || cfg.Timeout.Seconds() != 4 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestWriteEnvValuesPreservesKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("# keep\nOTHER=value\nECOLOGY_JSESSIONID=old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WriteEnvValues(path, map[string]string{"ECOLOGY_JSESSIONID": "new", "ECOLOGY_USERNAME": "alice"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if text == "" || !containsAll(text, "OTHER=value", "ECOLOGY_JSESSIONID=new", "ECOLOGY_USERNAME=alice") {
		t.Fatalf("unexpected env content: %s", text)
	}
}

func containsAll(text string, values ...string) bool {
	for _, value := range values {
		if !strings.Contains(text, value) {
			return false
		}
	}
	return true
}
