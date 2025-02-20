package generate

//go:generate buf generate --template {"version":"v2","plugins":[{"remote":"buf.build/protocolbuffers/go:v1.34.2","out":".","opt":["paths=source_relative"]}]}
//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate ./internal/data/ent/schema --feature sql/lock
//go:generate go run -mod=mod github.com/google/wire/cmd/wire ./cmd/server
//go:generate go run -mod=mod github.com/google/wire/cmd/wire ./test
