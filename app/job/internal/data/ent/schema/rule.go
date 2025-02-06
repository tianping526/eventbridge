package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

type Rule struct {
	ent.Schema
}

func (Rule) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
	}
}

func (Rule) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDMixin{},
		mixin.Time{},
	}
}

func (Rule) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(64).
			Comment("rule name"),
		field.String("bus_name").
			MaxLen(64).
			Comment("event bus name"),
		field.Uint8("status").
			Comment("rule status, 1-enabled, 2-disabled"),
		field.String("pattern").
			MaxLen(1024).
			Comment("rule pattern"),
		field.String("targets").
			MaxLen(4096).
			Comment("rule targets"),
	}
}

func (Rule) Edges() []ent.Edge {
	return []ent.Edge{}
}

func (Rule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("bus_name", "name").Unique(),
	}
}
