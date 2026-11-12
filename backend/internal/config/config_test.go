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
	if c.ClientIPHeader != "X-Real-IP" || c.CORSOrigins != nil || c.BindAddr != "" || c.ListenAddr() != ":8080" {
		t.Fatalf("client IP header %q, CORS %v, listen %q", c.ClientIPHeader, c.CORSOrigins, c.ListenAddr())
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
		"rate_limit_invalid":      func(e map[string]string) { e["RATE_LIMIT_CREATE_PER_MIN"] = "0" },
		"production_without_cors": func(e map[string]string) { e["APP_ENV"] = "production" },
		"client_ip_xff":           func(e map[string]string) { e["CLIENT_IP_HEADER"] = "X-Forwarded-For" },
		"client_ip_xff_lower":     func(e map[string]string) { e["CLIENT_IP_HEADER"] = "x-forwarded-for" },
		"client_ip_forwarded":     func(e map[string]string) { e["CLIENT_IP_HEADER"] = "Forwarded" },
		"client_ip_list":          func(e map[string]string) { e["CLIENT_IP_HEADER"] = "X-Real-IP,CF-Connecting-IP" },
		"client_ip_space":         func(e map[string]string) { e["CLIENT_IP_HEADER"] = "CF Connecting IP" },
		"client_ip_colon":         func(e map[string]string) { e["CLIENT_IP_HEADER"] = "X-Real-IP:" },
		"bind_not_ip":             func(e map[string]string) { e["BIND_ADDR"] = "localhost" },
		"bind_with_port":          func(e map[string]string) { e["BIND_ADDR"] = "127.0.0.1:8080" },
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

func TestProductionDefaults(t *testing.T) {
	env := baseEnv()
	env["APP_ENV"] = "production"
	env["CORS_ALLOWED_ORIGINS"] = "https://sakunoie.example"
	c, err := load(env)
	if err != nil {
		t.Fatal(err)
	}
	if c.DocsEnabled() {
		t.Fatal("docs must be disabled in production")
	}
	// Loopback by default: only cloudflared on the same host reaches the API.
	if c.BindAddr != "127.0.0.1" || c.ListenAddr() != "127.0.0.1:8080" {
		t.Fatalf("production listen address %q", c.ListenAddr())
	}
	env["BIND_ADDR"] = "0.0.0.0"
	if c, err = load(env); err != nil || c.ListenAddr() != "0.0.0.0:8080" {
		t.Fatalf("explicit BIND_ADDR: %v %v", c, err)
	}
}

func TestCORSOriginValidation(t *testing.T) {
	bad := []string{
		"*", "null", "https://*.sakunoie.example", "sakunoie.example", "//sakunoie.example",
		"https://sakunoie.example/", "https://sakunoie.example/app", "https://sakunoie.example?x=1",
		"https://sakunoie.example#f", "https://user@sakunoie.example", "ftp://sakunoie.example",
		"HTTPS://sakunoie.example", "https://Sakunoie.example", "https://sakunoie.example:443",
		"http://127.0.0.1:80", "https://", "https://:8443",
		"https://ok.example,*",
	}
	for _, o := range bad {
		env := baseEnv()
		env["CORS_ALLOWED_ORIGINS"] = o
		if _, err := load(env); err == nil {
			t.Errorf("%q: expected refusal", o)
		}
	}
	env := baseEnv()
	env["CORS_ALLOWED_ORIGINS"] = " https://sakunoie.example , http://127.0.0.1:3100,http://[::1]:3100 "
	c, err := load(env)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(c.CORSOrigins, " ") != "https://sakunoie.example http://127.0.0.1:3100 http://[::1]:3100" {
		t.Fatalf("origins: %v", c.CORSOrigins)
	}
}

func TestClientIPHeaderAndBindAddr(t *testing.T) {
	env := baseEnv()
	env["CLIENT_IP_HEADER"] = "CF-Connecting-IP"
	env["BIND_ADDR"] = "::1"
	env["PORT"] = "8181"
	c, err := load(env)
	if err != nil {
		t.Fatal(err)
	}
	if c.ClientIPHeader != "CF-Connecting-IP" || c.ListenAddr() != "[::1]:8181" {
		t.Fatalf("header %q, listen %q", c.ClientIPHeader, c.ListenAddr())
	}
}
