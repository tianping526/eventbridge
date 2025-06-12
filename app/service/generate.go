package service

//go:generate go tool buf generate --template {"version":"v2","plugins":[{"remote":"buf.build/protocolbuffers/go:v1.34.2","out":".","opt":["paths=source_relative"]}]}
//go:generate go tool ent generate ./internal/data/ent/schema --feature sql/lock
//go:generate go tool wire ./cmd/server
//go:generate go tool wire ./test
