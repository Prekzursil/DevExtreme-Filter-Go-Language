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
// relative to the execution path of the main application. It's a ``var`` so
// tests can override it via ``t.TempDir()``.
var SchemaDefinitionsDir = "./schema_definitions"

type SchemaFieldDefinition struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

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

func GenerateGoAdapterCode(req SchemaRequest) (string, error) {
	if req.EntityName == "" {
		return "", fmt.Errorf("entity name cannot be empty for adapter generation")
	}

	sanitizedEntityTypeName := req.EntityName
	sanitizedEntityTypeName = strings.ReplaceAll(sanitizedEntityTypeName, "-", "")
	sanitizedEntityTypeName = strings.ReplaceAll(sanitizedEntityTypeName, "_", "")
	sanitizedEntityTypeName = strings.ReplaceAll(sanitizedEntityTypeName, " ", "")
	if len(sanitizedEntityTypeName) == 0 {
		return "", fmt.Errorf("sanitized entity name is empty for adapter")
	}
	if len(sanitizedEntityTypeName) > 0 && unicode.IsLower(rune(sanitizedEntityTypeName[0])) {
		runes := []rune(sanitizedEntityTypeName)
		runes[0] = unicode.ToUpper(runes[0])
		sanitizedEntityTypeName = string(runes)
	}

	entityNameLower := strings.ToLower(sanitizedEntityTypeName)
	if _, isKeyword := GoKeywords[entityNameLower]; isKeyword {
		entityNameLower += "_"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("package main // Or your appropriate package\n\n"))
	sb.WriteString("import (\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"strings\"\n")
	sb.WriteString("\t\"time\"\n\n")
	sb.WriteString(fmt.Sprintf("\t\"transaction-filter-backend/ent/%s\"\n", entityNameLower))
	sb.WriteString(fmt.Sprintf("\t\"transaction-filter-backend/ent/predicate\" // For predicate.%s type alias\n", sanitizedEntityTypeName))
	sb.WriteString("\t\"entgo.io/ent/dialect/sql\" \n")
	sb.WriteString(")\n\n")

	adapterName := fmt.Sprintf("%sAdapter", sanitizedEntityTypeName)
	sb.WriteString(fmt.Sprintf("// %s implements the EntityAdapter for the %s entity.\n", adapterName, sanitizedEntityTypeName))
	sb.WriteString(fmt.Sprintf("type %s struct{}\n\n", adapterName))

	sb.WriteString(fmt.Sprintf("// GetPredicateForField constructs a predicate for %s.\n", sanitizedEntityTypeName))
	sb.WriteString(fmt.Sprintf("func (ta *%s) GetPredicateForField(field string, op string, val interface{}) (PredicateFunc, error) {\n", adapterName))
	sb.WriteString("\tfield = strings.ToLower(field)\n")
	sb.WriteString("\tswitch field {\n")
	for _, f := range req.Fields {
		goFieldName := f.Name

		sb.WriteString(fmt.Sprintf("\tcase \"%s\":\n", strings.ToLower(f.Name)))
		sb.WriteString(fmt.Sprintf("\t\t// TODO: Implement predicate logic for field '%s' (type: %s)\n", f.Name, f.Type))
		sb.WriteString(fmt.Sprintf("\t\t// Example for string EQ: return PredicateFunc(%s.%sEQ(val.(string))), nil\n", entityNameLower, goFieldName))
		sb.WriteString(fmt.Sprintf("\t\t// Example for int GT: return PredicateFunc(%s.%sGT(val.(int))), nil\n", entityNameLower, goFieldName))
		sb.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"predicate for field '%s' (type %s) not fully implemented yet\")\n", f.Name, f.Type))
	}
	sb.WriteString("\tdefault:\n")
	sb.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"unsupported field for %s: %%s\", field)\n", sanitizedEntityTypeName))
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// GetAndPredicate combines multiple predicates with AND for %s.\n", sanitizedEntityTypeName))
	sb.WriteString(fmt.Sprintf("func (ta *%s) GetAndPredicate(predicates ...PredicateFunc) PredicateFunc {\n", adapterName))
	sb.WriteString("\tif len(predicates) == 0 {\n\t\treturn nil\n\t}\n")
	sb.WriteString(fmt.Sprintf("\tvar specificPredicates []predicate.%s\n", sanitizedEntityTypeName))
	sb.WriteString("\tfor _, p := range predicates {\n")
	sb.WriteString("\t\tif p != nil {\n")
	sb.WriteString(fmt.Sprintf("\t\t\tspecificPredicates = append(specificPredicates, predicate.%s(p))\n", sanitizedEntityTypeName))
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tif len(specificPredicates) == 0 {\n\t\treturn nil\n\t}\n")
	sb.WriteString(fmt.Sprintf("\treturn PredicateFunc(%s.And(specificPredicates...))\n", entityNameLower))
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// GetOrPredicate combines multiple predicates with OR for %s.\n", sanitizedEntityTypeName))
	sb.WriteString(fmt.Sprintf("func (ta *%s) GetOrPredicate(predicates ...PredicateFunc) PredicateFunc {\n", adapterName))
	sb.WriteString("\tif len(predicates) == 0 {\n\t\treturn nil\n\t}\n")
	sb.WriteString(fmt.Sprintf("\tvar specificPredicates []predicate.%s\n", sanitizedEntityTypeName))
	sb.WriteString("\tfor _, p := range predicates {\n")
	sb.WriteString("\t\tif p != nil {\n")
	sb.WriteString(fmt.Sprintf("\t\t\tspecificPredicates = append(specificPredicates, predicate.%s(p))\n", sanitizedEntityTypeName))
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tif len(specificPredicates) == 0 {\n\t\treturn nil\n\t}\n")
	sb.WriteString(fmt.Sprintf("\treturn PredicateFunc(%s.Or(specificPredicates...))\n", entityNameLower))
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// GetNotPredicate negates a predicate for %s.\n", sanitizedEntityTypeName))
	sb.WriteString(fmt.Sprintf("func (ta *%s) GetNotPredicate(p PredicateFunc) PredicateFunc {\n", adapterName))
	sb.WriteString("\tif p == nil { return nil }\n")
	// This is the critical line, ensuring it's a single, correct Sprintf call.
	sb.WriteString(fmt.Sprintf("\treturn PredicateFunc(%s.Not(predicate.%s(p)))\n", entityNameLower, sanitizedEntityTypeName))
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("func init() {\n"))
	sb.WriteString(fmt.Sprintf("\t// Ensure this adapter is registered. The entity name should be lowercase.\n"))
	sb.WriteString(fmt.Sprintf("\t// Note: You might need to make RegisterAdapter public if it's in another package,\n"))
	sb.WriteString(fmt.Sprintf("\t// or call this registration from your main package.\n"))
	sb.WriteString(fmt.Sprintf("\t// RegisterAdapter(\"%s\", &%s{})\n", entityNameLower, adapterName))
	sb.WriteString("}\n")

	return sb.String(), nil
}
