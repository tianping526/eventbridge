package event

import (
	"context"

	"github.com/tianping526/eventbridge/app/internal/rule"
)

type Handler func(context.Context, *rule.EventExt) (interface{}, error)

type Receiver interface {
	Receive(ctx context.Context, handler Handler) error
	Close(ctx context.Context) error
}
