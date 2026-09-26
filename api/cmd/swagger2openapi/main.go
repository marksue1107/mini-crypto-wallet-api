// Command swagger2openapi converts the swag-generated Swagger 2.0 spec
// (docs/swagger.json) into an OpenAPI 3.0 spec (docs/openapi.yaml).
//
// swag (github.com/swaggo/swag) only emits Swagger 2.0. Some frontend
// codegen tools (e.g. openapi-typescript) require OpenAPI 3.0, so this
// conversion step exists to bridge the gap. See docs/AUDIT.md A2.
//
// Usage (from the api/ directory, after running `swag init`):
//
//	go run ./cmd/swagger2openapi
package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"gopkg.in/yaml.v3"
)

const (
	inputPath  = "docs/swagger.json"
	outputPath = "docs/openapi.yaml"
)

func main() {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		log.Fatalf("❌ failed to read %s (run `swag init -g main/main.go -o docs` first): %v", inputPath, err)
	}

	var doc2 openapi2.T
	if err := json.Unmarshal(data, &doc2); err != nil {
		log.Fatalf("❌ failed to parse %s as Swagger 2.0: %v", inputPath, err)
	}

	doc3, err := openapi2conv.ToV3(&doc2)
	if err != nil {
		log.Fatalf("❌ failed to convert to OpenAPI 3.0: %v", err)
	}

	jsonBytes, err := json.Marshal(doc3)
	if err != nil {
		log.Fatalf("❌ failed to marshal converted spec: %v", err)
	}

	// Round-trip through a generic map so yaml.v3 respects the JSON key
	// names (openapi3.T's struct tags are `json:"..."`, which yaml.v3
	// doesn't understand on its own).
	var generic any
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		log.Fatalf("❌ failed to re-parse converted spec: %v", err)
	}

	yamlBytes, err := yaml.Marshal(generic)
	if err != nil {
		log.Fatalf("❌ failed to marshal spec as YAML: %v", err)
	}

	if err := os.WriteFile(outputPath, yamlBytes, 0o644); err != nil {
		log.Fatalf("❌ failed to write %s: %v", outputPath, err)
	}

	log.Printf("✅ wrote %s (OpenAPI %s)", outputPath, doc3.OpenAPI)
}
