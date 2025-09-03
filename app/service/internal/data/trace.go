package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSDK "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/tianping526/eventbridge/app/service/internal/conf"
)

func NewTracerProvider(
	ai *conf.AppInfo, conf *conf.Bootstrap, l log.Logger,
) (trace.TracerProvider, func(), error) {
	logger := log.NewHelper(log.With(l, "module", "data/trace", "caller", log.DefaultCaller))
	if conf.Trace == nil {
		return noop.NewTracerProvider(), func() {}, nil
	}
	exp, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpointURL(conf.Trace.EndpointUrl),
	)
	if err != nil {
		return nil, nil, err
	}
	tp := traceSDK.NewTracerProvider(
		traceSDK.WithBatcher(exp),
		traceSDK.WithResource(resource.NewSchemaless(
			semConv.ServiceNameKey.String(ai.Name),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, func() {
		err2 := tp.Shutdown(context.Background())
		if err2 != nil {
			logger.Errorf("error shutting down tracer provider: %v", err2)
		}
	}, nil
}
