// Command openapi writes the committed contract api/openapi.yaml from the
// same Huma configuration as the server. Run from backend/:
//
//	go run ./cmd/openapi
package main

import (
	"flag"
	"log"
	"os"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/routes"
)

func main() {
	out := flag.String("out", "api/openapi.yaml", "output path")
	flag.Parse()
	models.InstallErrorConstructors()
	b, err := routes.OpenAPIYAML()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		log.Fatal(err)
	}
}
