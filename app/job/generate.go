package generate

//go:generate go tool buf generate --template {"version":"v2","plugins":[{"local":["go","run","google.golang.org/protobuf/cmd/protoc-gen-go"],"out":".","opt":["paths=source_relative"]}]}
//go:generate go tool ent generate ./internal/data/ent/schema --feature sql/lock
//go:generate go tool wire ./cmd/server
//go:generate go tool wire ./test
