package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

var testKeyB64 = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))

func baseEnv() map[string]string {
	return map[string]string{
		"APP_ENV":                    "test",
		"GORM_DATABASE_URL":          "root:test@tcp(127.0.0.1:33306)/lodging_test",
		"PASSPORT_ENC_KEYS":          "1:" + testKeyB64,
		"PASSPORT_ENC_ACTIVE_KEY_ID": "1",
		"IDEMPOTENCY_HMAC_KEY":       testKeyB64,
		"FIREBASE_PROJECT_ID":        "demo-sakunoie",
	}
}

func load(env map[string]string) (*Config, error) {
	return Load(func(k string) string { return env[k] })
}

func TestLoadDefaults(t *testing.T) {
	c, err := load(baseEnv())
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != "8080" || c.RateLimitCreatePerMin != 10 || c.RateLimitManagePerMin != 60 ||
		c.RateLimitSurveyPerMin != 10 || c.SurveyMinResponses != 5 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
	if strings.Join(c.TrustedProxies, ",") != "127.0.0.1,::1" {
		t.Fatalf("trusted proxies: %v", c.TrustedProxies)
	}
	if !c.DocsEnabled() {
		t.Fatal("docs must be enabled in test")
	}
}

func TestLoadRefusals(t *testing.T) {
	cases := map[string]func(map[string]string){
		"app_env_unset":   func(e map[string]string) { delete(e, "APP_ENV") },
		"app_env_unknown": func(e map[string]string) { e["APP_ENV"] = "staging" },
		"emulator_in_production": func(e map[string]string) {
			e["APP_ENV"] = "production"
			e["FIREBASE_PROJECT_ID"] = "demo-sakunoie"
			e["FIREBASE_AUTH_EMULATOR_HOST"] = "127.0.0.1:9099"
		},
		"emulator_non_demo": func(e map[string]string) {
			e["FIREBASE_PROJECT_ID"] = "real-project"
			e["FIREBASE_AUTH_EMULATOR_HOST"] = "127.0.0.1:9099"
		},
		"firebase_project_missing": func(e map[string]string) { delete(e, "FIREBASE_PROJECT_ID") },
		"db_missing":               func(e map[string]string) { delete(e, "GORM_DATABASE_URL") },
		"passport_missing":         func(e map[string]string) { delete(e, "PASSPORT_ENC_KEYS") },
		"passport_short": func(e map[string]string) {
			e["PASSPORT_ENC_KEYS"] = "1:" + base64.StdEncoding.EncodeToString([]byte("short"))
		},
		"active_missing": func(e map[string]string) { delete(e, "PASSPORT_ENC_ACTIVE_KEY_ID") },
		"active_unknown": func(e map[string]string) { e["PASSPORT_ENC_ACTIVE_KEY_ID"] = "2" },
		"hmac_missing":   func(e map[string]string) { delete(e, "IDEMPOTENCY_HMAC_KEY") },
		"hmac_short": func(e map[string]string) {
			e["IDEMPOTENCY_HMAC_KEY"] = base64.StdEncoding.EncodeToString([]byte("short"))
		},
		"rate_limit_invalid": func(e map[string]string) { e["RATE_LIMIT_CREATE_PER_MIN"] = "0" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			env := baseEnv()
			mutate(env)
			_, err := load(env)
			if err == nil {
				t.Fatal("expected refusal")
			}
			if strings.Contains(err.Error(), testKeyB64) {
				t.Fatal("error leaks key material")
			}
		})
	}
}

func TestEmulatorAllowedOutsideProduction(t *testing.T) {
	env := baseEnv()
	env["FIREBASE_PROJECT_ID"] = "demo-sakunoie"
	env["FIREBASE_AUTH_EMULATOR_HOST"] = "127.0.0.1:9099"
	if _, err := load(env); err != nil {
		t.Fatal(err)
	}
}

func TestProductionDisablesDocs(t *testing.T) {
	env := baseEnv()
	env["APP_ENV"] = "production"
	c, err := load(env)
	if err != nil {
		t.Fatal(err)
	}
	if c.DocsEnabled() {
		t.Fatal("docs must be disabled in production")
	}
}
