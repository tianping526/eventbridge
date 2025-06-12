package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

type IDMixin struct {
	mixin.Schema
}

// Fields of the Mixin.
func (IDMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").
			Immutable(),
	}
}
