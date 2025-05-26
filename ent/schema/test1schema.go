package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Test1Schema holds the schema definition for the Test1Schema entity.
// It's a test entity with 6 fields of various types.
type Test1Schema struct {
	ent.Schema
}

// Fields of the Test1Schema.
func (Test1Schema) Fields() []ent.Field {
	return []ent.Field{
		field.String("field_string").Default("default string"),
		field.Int("field_int").Default(0),
		field.Float("field_float").Default(0.0),
		field.Bool("field_bool").Default(false),
		field.Time("field_time").Default(time.Now),
		field.Text("field_text").Optional(), // Using Text for potentially longer string
	}
}

// Edges of the Test1Schema.
func (Test1Schema) Edges() []ent.Edge {
	return nil
}
