package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type Version struct {
	ent.Schema
}

// Annotations of the Version.
func (Version) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
	}
}

// Mixin of the Version.
func (Version) Mixin() []ent.Mixin {
	return []ent.Mixin{}
}

// Fields of the Version.
func (Version) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("version").
			Comment("version of the resource"),
	}
}

// Edges of the Version.
func (Version) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the Version.
func (Version) Indexes() []ent.Index {
	return []ent.Index{}
}
