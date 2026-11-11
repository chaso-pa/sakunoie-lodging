// Package api holds the committed OpenAPI contract and its drift test.
package api

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/routes"
	"github.com/goccy/go-yaml"
)

func TestOpenAPIDrift(t *testing.T) {
	generated, err := routes.OpenAPIYAML()
	if err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, committed) {
		t.Fatal("api/openapi.yaml is stale: run `go run ./cmd/openapi` from backend/ and commit the result")
	}
}

func TestOpenAPIContract(t *testing.T) {
	b, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Fatalf("openapi = %v, want 3.1.0", doc["openapi"])
	}
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	env := schemas["ErrorEnvelope"].(map[string]any)
	var props []string
	for k := range env["properties"].(map[string]any) {
		props = append(props, k)
	}
	slices.Sort(props)
	if strings.Join(props, ",") != "code,details,message,request_id" {
		t.Fatalf("ErrorEnvelope properties = %v", props)
	}
	if _, ok := schemas["ErrorModel"]; ok {
		t.Fatal("Huma's default ErrorModel must not appear")
	}

	paths := doc["paths"].(map[string]any)
	for _, want := range []string{
		"get /api/v1/facilities/{code}/public",
		"post /api/v1/guest-registrations",
		"get /api/v1/guest-registrations/manage",
		"patch /api/v1/guest-registrations/manage",
		"post /api/v1/guest-registrations/manage/guests",
		"patch /api/v1/guest-registrations/manage/guests/{guest_public_id}",
		"post /api/v1/surveys",
		"get /api/v1/admin/facilities",
		"get /api/v1/admin/guest-registrations",
		"get /api/v1/admin/guest-registrations/{id}",
		"patch /api/v1/admin/guest-registrations/{id}",
		"post /api/v1/admin/guest-registrations/{id}/verify-identity",
		"post /api/v1/admin/guest-registrations/{id}/reissue-edit-link",
		"get /api/v1/admin/guest-registrations/export",
		"get /api/v1/admin/surveys/summary",
	} {
		method, path, _ := strings.Cut(want, " ")
		ops, ok := paths[path].(map[string]any)
		if !ok || ops[method] == nil {
			t.Errorf("missing operation %s", want)
		}
	}

	// Every 4xx/5xx response references ErrorEnvelope.
	for path, item := range paths {
		for method, op := range item.(map[string]any) {
			for code, resp := range op.(map[string]any)["responses"].(map[string]any) {
				if code[0] != '4' && code[0] != '5' && code != "default" {
					continue
				}
				js, _ := json.Marshal(resp)
				if !strings.Contains(string(js), "#/components/schemas/ErrorEnvelope") {
					t.Errorf("%s %s %s does not reference ErrorEnvelope", method, path, code)
				}
			}
		}
	}
	// Every admin operation requires the adminBearer scheme.
	for path, item := range paths {
		if !strings.HasPrefix(path, "/api/v1/admin/") {
			continue
		}
		for method, op := range item.(map[string]any) {
			js, _ := json.Marshal(op.(map[string]any)["security"])
			if string(js) != `[{"adminBearer":[]}]` {
				t.Errorf("%s %s security = %s", method, path, js)
			}
		}
	}
	if strings.Contains(string(b), "$schema") {
		t.Error("spec must not contain $schema link fields")
	}
}
