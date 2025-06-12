package server

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"go.opentelemetry.io/otel/trace"

	"github.com/tianping526/eventbridge/app/internal/event"
	"github.com/tianping526/eventbridge/app/job/internal/biz"
	"github.com/tianping526/eventbridge/app/job/internal/data"
)

// NewEventServer new an event server.
func NewEventServer(
	logger log.Logger,
	m *data.Metric,
	r event.Receiver,
	ec *biz.EventUseCase,
	_ trace.TracerProvider, // otel.SetTracerProvider(provider) instead, but need to declare to wire injection
) *event.Server {
	opts := []event.ServerOption{
		event.WithMiddleware(
			recovery.Recovery(),
			tracing.Server(),
			metrics.Server(
				metrics.WithSeconds(m.ServerDurationSec),
				metrics.WithRequests(m.ServerCodeTotal),
			),
			recovery.Recovery(),
			logging.Server(logger),
		),
		event.WithTimeout(0), // no timeout, set by receiver
		event.WithOperation("Consumer"),
	}

	srv := event.NewServer(r, ec.HandleEvent, opts...)
	return srv
}
