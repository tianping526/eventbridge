package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

type EventSchema struct {
	ent.Schema
}

func (EventSchema) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
	}
}

func (EventSchema) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDMixin{},
		mixin.Time{},
	}
}

func (EventSchema) Fields() []ent.Field {
	return []ent.Field{
		field.String("source").
			MaxLen(64).
			Comment("source of the event"),
		field.String("type").
			MaxLen(64).
			Comment("type of the event"),
		field.String("bus_name").
			MaxLen(64).
			Comment("event bus name"),
		field.String("spec").
			MaxLen(1024).
			Comment("JSON schema of the event"),
		field.Uint32("version").
			Comment("version of the event schema"),
	}
}

func (EventSchema) Edges() []ent.Edge {
	return []ent.Edge{}
}

func (EventSchema) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source", "type").Unique(),
	}
}
