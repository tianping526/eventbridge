package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

type Bus struct {
	ent.Schema
}

func (Bus) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
	}
}

func (Bus) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDMixin{},
		mixin.Time{},
	}
}

func (Bus) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(64).
			Comment("event bus name"),
		field.Uint8("mode").
			Comment("event bus mode. 1-concurrently, 2-orderly"),
		field.String("source_topic").
			MaxLen(128).
			Comment("source event topic name"),
		field.String("source_delay_topic").
			MaxLen(128).
			Comment("source event delay topic name"),
		field.String("target_exp_decay_topic").
			MaxLen(128).
			Comment("target event exponential decay topic name"),
		field.String("target_backoff_topic").
			MaxLen(128).
			Comment("target event backoff topic name"),
	}
}

func (Bus) Edges() []ent.Edge {
	return []ent.Edge{}
}

func (Bus) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").Unique(),
	}
}
