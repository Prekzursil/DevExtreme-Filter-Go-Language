package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Test2Schema holds the schema definition for the Test2Schema entity.
// It's a test entity with 8 fields of various types.
type Test2Schema struct {
	ent.Schema
}

// Fields of the Test2Schema.
func (Test2Schema) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Default("Unknown Name"),
		field.Text("description").Optional(),
		field.Int("quantity").Default(0),
		field.Float("price").Default(0.0),
		field.Bool("active").Default(true),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.String("item_type").Optional(),
	}
}

// Edges of the Test2Schema.
func (Test2Schema) Edges() []ent.Edge {
	return nil
}
