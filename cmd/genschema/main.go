// Package main generates kindplane.schema.json from the config package types.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/kanzi/kindplane/internal/config"
)

const (
	defaultSchemaID = "https://raw.githubusercontent.com/kanzifucius/kindplane/main/kindplane.schema.json"
	schemaDraft     = "http://json-schema.org/draft-07/schema#"
	defaultOutput   = "kindplane.schema.json"
	ghPagesBase     = "https://kanzifucius.github.io/kindplane"
)

func main() {
	var version = flag.String("version", "", "Version for schema $id (e.g., '1.0.0', 'dev', 'latest'). If empty, uses raw GitHub URL")
	var output = flag.String("output", defaultOutput, "Output file path")
	flag.Parse()

	// Determine schema ID based on version
	var schemaID string
	if *version != "" {
		// Use GitHub Pages URL for versioned schemas
		schemaID = fmt.Sprintf("%s/%s/kindplane.schema.json", ghPagesBase, *version)
	} else {
		// Default to raw GitHub URL for backward compatibility
		schemaID = defaultSchemaID
	}

	r := &jsonschema.Reflector{
		AllowAdditionalProperties: false,
		ExpandedStruct:            true,
		FieldNameTag:              "yaml",
		BaseSchemaID:              jsonschema.ID(schemaID),
		LookupComment:             lookupConfigComment,
	}

	schema := r.Reflect(&config.Config{})
	if schema == nil {
		log.Fatal("Reflect returned nil schema")
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("Marshal schema: %v", err)
	}

	// Post-process: draft-07 uses "definitions" and different $schema
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Fatalf("Unmarshal for post-process: %v", err)
	}

	raw["$schema"] = schemaDraft
	raw["$id"] = schemaID
	raw["title"] = "Kindplane Configuration"
	raw["description"] = "Configuration schema for kindplane.yaml - a tool for bootstrapping Kind clusters with Crossplane"

	// Rename $defs to definitions for draft-07 compatibility
	if defs, ok := raw["$defs"].(map[string]interface{}); ok {
		raw["definitions"] = defs
		delete(raw, "$defs")
	}

	// Fix $ref paths: #/$defs/ -> #/definitions/
	data, err = json.MarshalIndent(raw, "", "  ")
	if err != nil {
		log.Fatalf("Marshal after post-process: %v", err)
	}
	s := string(data)
	s = strings.ReplaceAll(s, "#/$defs/", "#/definitions/")
	data = []byte(s)

	if err := os.WriteFile(*output, data, 0644); err != nil {
		log.Fatalf("Write %s: %v", *output, err)
	}
	fmt.Printf("Wrote %s (schema $id: %s)\n", *output, schemaID)
}

// lookupConfigComment returns description from config struct "comment" and "doc" tags.
func lookupConfigComment(t reflect.Type, fieldName string) string {
	if fieldName == "" {
		return ""
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Name == fieldName {
			comment := f.Tag.Get("comment")
			doc := f.Tag.Get("doc")
			if doc != "" {
				if comment != "" {
					return comment + "\n" + doc
				}
				return doc
			}
			return comment
		}
	}
	return ""
}
