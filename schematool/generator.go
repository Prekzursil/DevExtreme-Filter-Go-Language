// Package schematool provides HTTP handlers and code generators for dynamic schema management.
package schematool

import (
	"fmt"
	"strings"
	"unicode"
	// "encoding/json" // Not used directly in this file
	// "os"            // Not used directly in this file
	// "path/filepath" // Not used directly in this file
)

// GoKeywords is a list of Go reserved keywords.
var GoKeywords = map[string]bool{
	"break":       true,
	"default":     true,
	"func":        true,
	"interface":   true,
	"select":      true,
	"case":        true,
	"defer":       true,
	"go":          true,
	"map":         true,
	"struct":      true,
	"chan":        true,
	"else":        true,
	"goto":        true,
	"package":     true,
	"switch":      true,
	"const":       true,
	"fallthrough": true,
	"if":          true,
	"range":       true,
	"type":        true,
	"continue":    true,
	"for":         true,
	"import":      true,
	"return":      true,
	"var":         true,
}

// SchemaDefinitionsDir is the directory where schema JSON files are saved,
// relative to the execution path of the main application. It's a “var“ so
// tests can override it via “t.TempDir()“.
var SchemaDefinitionsDir = "./schema_definitions"

// SchemaFieldDefinition is a single field (name + Go type) of a dynamic
// entity schema.
type SchemaFieldDefinition struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// SchemaRequest is the payload describing an entity to generate: its name and
// the list of field definitions.
type SchemaRequest struct {
	EntityName string                  `json:"entityName"`
	Fields     []SchemaFieldDefinition `json:"fields"`
}

// entFieldEmitters maps a SchemaFieldDefinition.Type to the corresponding
// entgo schema field constructor. Lookup-table dispatch replaces a 5-branch
// switch and drops GenerateGoSchemaCode below qlty's "many returns" threshold.
var entFieldEmitters = map[string]string{
	"string":    "field.String",
	"int":       "field.Int",
	"bool":      "field.Bool",
	"time.Time": "field.Time",
	"float64":   "field.Float",
}

// GenerateGoSchemaCode validates req and returns the Go source for an entgo
// ent.Schema definition of the requested entity.
func GenerateGoSchemaCode(req SchemaRequest) (string, error) {
	name, err := validateAndSanitizeSchemaRequest(req)
	if err != nil {
		return "", err
	}
	return assembleSchemaSource(name, req.Fields)
}

func validateAndSanitizeSchemaRequest(req SchemaRequest) (string, error) {
	if req.EntityName == "" {
		return "", fmt.Errorf("entity name cannot be empty")
	}
	if len(req.Fields) == 0 {
		return "", fmt.Errorf("at least one field is required")
	}
	name := sanitizeEntityName(req.EntityName)
	if name == "" {
		return "", fmt.Errorf("sanitized entity name is empty")
	}
	return name, nil
}

func sanitizeEntityName(raw string) string {
	s := strings.ReplaceAll(raw, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, " ", "")
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if unicode.IsLower(runes[0]) {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}

func hasTimeField(fields []SchemaFieldDefinition) bool {
	for _, f := range fields {
		if f.Type == "time.Time" {
			return true
		}
	}
	return false
}

func assembleSchemaSource(name string, fields []SchemaFieldDefinition) (string, error) {
	var sb strings.Builder
	sb.WriteString("package schema\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"entgo.io/ent\"\n")
	sb.WriteString("\t\"entgo.io/ent/schema/field\"\n")
	if hasTimeField(fields) {
		sb.WriteString("\t\"time\"\n")
	}
	sb.WriteString(")\n\n")

	fmt.Fprintf(&sb, "// %s holds the schema definition for the %s entity.\n", name, name)
	fmt.Fprintf(&sb, "type %s struct {\n\tent.Schema\n}\n\n", name)

	fmt.Fprintf(&sb, "// Fields of the %s.\n", name)
	fmt.Fprintf(&sb, "func (%s) Fields() []ent.Field {\n", name)
	sb.WriteString("\treturn []ent.Field{\n")

	if err := emitEntFields(&sb, fields); err != nil {
		return "", err
	}

	sb.WriteString("\t}\n}\n\n")

	fmt.Fprintf(&sb, "// Edges of the %s.\n", name)
	fmt.Fprintf(&sb, "func (%s) Edges() []ent.Edge {\n\treturn nil\n}\n", name)
	return sb.String(), nil
}

func emitEntFields(sb *strings.Builder, fields []SchemaFieldDefinition) error {
	for _, f := range fields {
		if f.Name == "" || f.Type == "" {
			return fmt.Errorf("field name and type cannot be empty (field: %+v)", f)
		}
		emitter, ok := entFieldEmitters[f.Type]
		if !ok {
			return fmt.Errorf("unsupported field type: %s for field %s", f.Type, f.Name)
		}
		fmt.Fprintf(sb, "\t\t%s(\"%s\"),\n", emitter, f.Name)
	}
	return nil
}

// GenerateGoAdapterCode returns the Go source for an EntityAdapter scaffold
// (predicate dispatch plus And/Or/Not combinators) for the requested entity.
func GenerateGoAdapterCode(req SchemaRequest) (string, error) {
	if req.EntityName == "" {
		return "", fmt.Errorf("entity name cannot be empty for adapter generation")
	}
	typeName := sanitizeEntityName(req.EntityName)
	if typeName == "" {
		return "", fmt.Errorf("sanitized entity name is empty for adapter")
	}
	lowerName := adapterEntityLower(typeName)
	adapterName := fmt.Sprintf("%sAdapter", typeName)

	var sb strings.Builder
	writeAdapterHeader(&sb, typeName, lowerName, adapterName)
	writeAdapterPredicateForField(&sb, typeName, lowerName, adapterName, req.Fields)
	writeAdapterCombinator(&sb, typeName, lowerName, adapterName, "And")
	writeAdapterCombinator(&sb, typeName, lowerName, adapterName, "Or")
	writeAdapterNotPredicate(&sb, typeName, lowerName, adapterName)
	writeAdapterInit(&sb, lowerName, adapterName)
	return sb.String(), nil
}

// adapterEntityLower lowercases the entity name and suffixes "_" when it
// collides with a Go keyword so the generated import alias stays valid.
func adapterEntityLower(typeName string) string {
	lower := strings.ToLower(typeName)
	if GoKeywords[lower] {
		lower += "_"
	}
	return lower
}

func writeAdapterHeader(sb *strings.Builder, typeName, lowerName, adapterName string) {
	sb.WriteString("package main // Or your appropriate package\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"strings\"\n")
	sb.WriteString("\t\"time\"\n\n")
	fmt.Fprintf(sb, "\t\"transaction-filter-backend/ent/%s\"\n", lowerName)
	fmt.Fprintf(sb, "\t\"transaction-filter-backend/ent/predicate\" // For predicate.%s type alias\n", typeName)
	sb.WriteString("\t\"entgo.io/ent/dialect/sql\" \n")
	sb.WriteString(")\n\n")

	fmt.Fprintf(sb, "// %s implements the EntityAdapter for the %s entity.\n", adapterName, typeName)
	fmt.Fprintf(sb, "type %s struct{}\n\n", adapterName)
}

func writeAdapterPredicateForField(sb *strings.Builder, typeName, lowerName, adapterName string, fields []SchemaFieldDefinition) {
	fmt.Fprintf(sb, "// GetPredicateForField constructs a predicate for %s.\n", typeName)
	fmt.Fprintf(sb, "func (ta *%s) GetPredicateForField(field string, op string, val interface{}) (PredicateFunc, error) {\n", adapterName)
	sb.WriteString("\tfield = strings.ToLower(field)\n")
	sb.WriteString("\tswitch field {\n")
	for _, f := range fields {
		fmt.Fprintf(sb, "\tcase \"%s\":\n", strings.ToLower(f.Name))
		fmt.Fprintf(sb, "\t\t// TODO: Implement predicate logic for field '%s' (type: %s)\n", f.Name, f.Type)
		fmt.Fprintf(sb, "\t\t// Example for string EQ: return PredicateFunc(%s.%sEQ(val.(string))), nil\n", lowerName, f.Name)
		fmt.Fprintf(sb, "\t\t// Example for int GT: return PredicateFunc(%s.%sGT(val.(int))), nil\n", lowerName, f.Name)
		fmt.Fprintf(sb, "\t\treturn nil, fmt.Errorf(\"predicate for field '%s' (type %s) not fully implemented yet\")\n", f.Name, f.Type)
	}
	sb.WriteString("\tdefault:\n")
	fmt.Fprintf(sb, "\t\treturn nil, fmt.Errorf(\"unsupported field for %s: %%s\", field)\n", typeName)
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")
}

// writeAdapterCombinator emits the GetAndPredicate / GetOrPredicate methods,
// which differ only in the logical operator ("And"/"Or") and the method name.
// Parameterising op collapses the two near-identical blocks into one.
func writeAdapterCombinator(sb *strings.Builder, typeName, lowerName, adapterName, op string) {
	fmt.Fprintf(sb, "// Get%sPredicate combines multiple predicates with %s for %s.\n", op, strings.ToUpper(op), typeName)
	fmt.Fprintf(sb, "func (ta *%s) Get%sPredicate(predicates ...PredicateFunc) PredicateFunc {\n", adapterName, op)
	sb.WriteString("\tif len(predicates) == 0 {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(sb, "\tvar specificPredicates []predicate.%s\n", typeName)
	sb.WriteString("\tfor _, p := range predicates {\n")
	sb.WriteString("\t\tif p != nil {\n")
	fmt.Fprintf(sb, "\t\t\tspecificPredicates = append(specificPredicates, predicate.%s(p))\n", typeName)
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tif len(specificPredicates) == 0 {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(sb, "\treturn PredicateFunc(%s.%s(specificPredicates...))\n", lowerName, op)
	sb.WriteString("}\n\n")
}

func writeAdapterNotPredicate(sb *strings.Builder, typeName, lowerName, adapterName string) {
	fmt.Fprintf(sb, "// GetNotPredicate negates a predicate for %s.\n", typeName)
	fmt.Fprintf(sb, "func (ta *%s) GetNotPredicate(p PredicateFunc) PredicateFunc {\n", adapterName)
	sb.WriteString("\tif p == nil { return nil }\n")
	fmt.Fprintf(sb, "\treturn PredicateFunc(%s.Not(predicate.%s(p)))\n", lowerName, typeName)
	sb.WriteString("}\n\n")
}

func writeAdapterInit(sb *strings.Builder, lowerName, adapterName string) {
	sb.WriteString("func init() {\n")
	sb.WriteString("\t// Ensure this adapter is registered. The entity name should be lowercase.\n")
	sb.WriteString("\t// Note: You might need to make RegisterAdapter public if it's in another package,\n")
	sb.WriteString("\t// or call this registration from your main package.\n")
	fmt.Fprintf(sb, "\t// RegisterAdapter(\"%s\", &%s{})\n", lowerName, adapterName)
	sb.WriteString("}\n")
}
