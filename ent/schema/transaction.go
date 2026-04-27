// Package schema defines the Ent schema models for in-memory test entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Transaction holds the schema definition for the Transaction entity.
type Transaction struct {
	ent.Schema
}

// Fields of the Transaction.
func (Transaction) Fields() []ent.Field {
	return []ent.Field{
		field.Time("date"),
		field.Float("amount"),
		field.String("name"),
		field.String("location"),
		field.String("category"),
		field.String("type"),
	}
}

// Edges of the Transaction.
func (Transaction) Edges() []ent.Edge {
	return nil
}
