package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeDotEnv(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadDotEnvDoesNotOverrideExisting(t *testing.T) {
	t.Setenv("SKIP_DOTENV", "")
	t.Setenv("SAKUNOIE_DOTENV_EXISTING", "from-env")
	t.Setenv("SAKUNOIE_DOTENV_NEW", "")
	os.Unsetenv("SAKUNOIE_DOTENV_NEW")
	p := writeDotEnv(t, "SAKUNOIE_DOTENV_EXISTING=from-file\nSAKUNOIE_DOTENV_NEW=from-file\n")

	loaded, err := LoadDotEnv(p)
	if err != nil || !loaded {
		t.Fatalf("loaded=%v err=%v", loaded, err)
	}
	if got := os.Getenv("SAKUNOIE_DOTENV_EXISTING"); got != "from-env" {
		t.Fatalf("existing variable overridden: %q", got)
	}
	if got := os.Getenv("SAKUNOIE_DOTENV_NEW"); got != "from-file" {
		t.Fatalf("new variable not loaded: %q", got)
	}
}

func TestLoadDotEnvMissingFile(t *testing.T) {
	t.Setenv("SKIP_DOTENV", "")
	loaded, err := LoadDotEnv(filepath.Join(t.TempDir(), "absent.env"))
	if err != nil || loaded {
		t.Fatalf("loaded=%v err=%v", loaded, err)
	}
}

func TestLoadDotEnvSkip(t *testing.T) {
	t.Setenv("SKIP_DOTENV", "1")
	t.Setenv("SAKUNOIE_DOTENV_SKIPPED", "")
	os.Unsetenv("SAKUNOIE_DOTENV_SKIPPED")
	p := writeDotEnv(t, "SAKUNOIE_DOTENV_SKIPPED=from-file\n")

	loaded, err := LoadDotEnv(p)
	if err != nil || loaded {
		t.Fatalf("loaded=%v err=%v", loaded, err)
	}
	if _, ok := os.LookupEnv("SAKUNOIE_DOTENV_SKIPPED"); ok {
		t.Fatal("SKIP_DOTENV=1 must not load the file")
	}
}
