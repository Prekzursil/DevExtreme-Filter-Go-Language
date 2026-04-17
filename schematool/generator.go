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
// relative to the execution path of the main application.
const SchemaDefinitionsDir = "./schema_definitions"

type SchemaFieldDefinition struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type SchemaRequest struct {
	EntityName string                  `json:"entityName"`
	Fields     []SchemaFieldDefinition `json:"fields"`
}

var entFieldBuilders = map[string]string{
	"string":    "field.String",
	"int":       "field.Int",
	"bool":      "field.Bool",
	"time.Time": "field.Time",
	"float64":   "field.Float",
}

func sanitizeEntityName(name string) (string, error) {
	n := strings.ReplaceAll(name, "-", "")
	n = strings.ReplaceAll(n, "_", "")
	n = strings.ReplaceAll(n, " ", "")
	if len(n) == 0 {
		return "", fmt.Errorf("sanitized entity name is empty")
	}
	if unicode.IsLower(rune(n[0])) {
		runes := []rune(n)
		runes[0] = unicode.ToUpper(runes[0])
		n = string(runes)
	}
	return n, nil
}

func hasTimeField(fields []SchemaFieldDefinition) bool {
	for _, f := range fields {
		if f.Type == "time.Time" {
			return true
		}
	}
	return false
}

func writeFieldEntries(sb *strings.Builder, fields []SchemaFieldDefinition) error {
	for _, f := range fields {
		if f.Name == "" || f.Type == "" {
			return fmt.Errorf("field name and type cannot be empty (field: %+v)", f)
		}
		builder, ok := entFieldBuilders[f.Type]
		if !ok {
			return fmt.Errorf("unsupported field type: %s for field %s", f.Type, f.Name)
		}
		sb.WriteString(fmt.Sprintf("\t\t%s(\"%s\"),\n", builder, f.Name))
	}
	return nil
}

func GenerateGoSchemaCode(req SchemaRequest) (string, error) {
	if req.EntityName == "" {
		return "", fmt.Errorf("entity name cannot be empty")
	}
	if len(req.Fields) == 0 {
		return "", fmt.Errorf("at least one field is required")
	}
	typeName, err := sanitizeEntityName(req.EntityName)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	writeSchemaImports(&sb, hasTimeField(req.Fields))
	writeSchemaStruct(&sb, typeName)
	sb.WriteString(fmt.Sprintf("// Fields of the %s.\n", typeName))
	sb.WriteString(fmt.Sprintf("func (%s) Fields() []ent.Field {\n", typeName))
	sb.WriteString("\treturn []ent.Field{\n")
	if err := writeFieldEntries(&sb, req.Fields); err != nil {
		return "", err
	}
	sb.WriteString("\t}\n}\n\n")
	writeEdgesStub(&sb, typeName)
	return sb.String(), nil
}

func writeSchemaImports(sb *strings.Builder, includeTime bool) {
	sb.WriteString("package schema\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"entgo.io/ent\"\n")
	sb.WriteString("\t\"entgo.io/ent/schema/field\"\n")
	if includeTime {
		sb.WriteString("\t\"time\"\n")
	}
	sb.WriteString(")\n\n")
}

func writeSchemaStruct(sb *strings.Builder, typeName string) {
	sb.WriteString(fmt.Sprintf("// %s holds the schema definition for the %s entity.\n", typeName, typeName))
	sb.WriteString(fmt.Sprintf("type %s struct {\n\tent.Schema\n}\n\n", typeName))
}

func writeEdgesStub(sb *strings.Builder, typeName string) {
	sb.WriteString(fmt.Sprintf("// Edges of the %s.\n", typeName))
	sb.WriteString(fmt.Sprintf("func (%s) Edges() []ent.Edge {\n\treturn nil\n}\n", typeName))
}

func GenerateGoAdapterCode(req SchemaRequest) (string, error) {
	if req.EntityName == "" {
		return "", fmt.Errorf("entity name cannot be empty for adapter generation")
	}
	sanitizedEntityTypeName, err := sanitizeEntityName(req.EntityName)
	if err != nil {
		return "", err
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
