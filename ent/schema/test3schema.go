package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Test3Schema holds the schema definition for the Test3Schema entity.
// It's a test entity with 11 fields of various types.
type Test3Schema struct {
	ent.Schema
}

// Fields of the Test3Schema.
func (Test3Schema) Fields() []ent.Field {
	return []ent.Field{
		field.String("sku").Unique().NotEmpty(),
		field.String("product_name").Default("Unnamed Product"),
		field.String("short_description").Optional(),
		field.Text("full_description").Optional(),
		field.Float("cost_price").Default(0.0),
		field.Float("retail_price").Default(0.0),
		field.Int("stock_count").Default(0),
		field.Bool("is_active").Default(true),
		field.Time("published_at").Optional(),
		field.Time("last_ordered_at").Optional(),
		field.String("tags").Optional().Comment("Comma-separated tags"),
	}
}

// Edges of the Test3Schema.
func (Test3Schema) Edges() []ent.Edge {
	return nil
}
