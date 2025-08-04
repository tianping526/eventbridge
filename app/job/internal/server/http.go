package server

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/gorilla/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/tianping526/eventbridge/app/job/internal/conf"
	"github.com/tianping526/eventbridge/app/job/internal/data"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	bc *conf.Bootstrap,
	logger log.Logger,
	m *data.Metric,
	_ trace.TracerProvider, // otel.SetTracerProvider(provider) instead, but need to declare to wire injection
) *http.Server {
	cs := bc.Server
	opts := []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			metrics.Server(
				metrics.WithSeconds(m.ServerDurationSec),
				metrics.WithRequests(m.ServerCodeTotal),
			),
			recovery.Recovery(),
			logging.Server(logger),
			ratelimit.Server(),
		),
		http.Filter(handlers.CORS(
			handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}),
			handlers.AllowedOrigins([]string{"*"}),
		)),
	}
	if cs.Http.Network != "" {
		opts = append(opts, http.Network(cs.Http.Network))
	}
	if cs.Http.Addr != "" {
		opts = append(opts, http.Address(cs.Http.Addr))
	}
	if cs.Http.Timeout != nil {
		opts = append(opts, http.Timeout(cs.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	srv.Handle("/metrics", promhttp.Handler())
	return srv
}
