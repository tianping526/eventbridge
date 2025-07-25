package data

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSDK "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/tianping526/eventbridge/app/job/internal/conf"
)

func NewTracerProvider(ai *conf.AppInfo, conf *conf.Bootstrap) (trace.TracerProvider, func(), error) {
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
		traceSDK.WithSampler(traceSDK.ParentBased(traceSDK.TraceIDRatioBased(1.0))),
	)
	otel.SetTracerProvider(tp)
	return tp, func() {
		err2 := tp.Shutdown(context.Background())
		if err2 != nil {
			fmt.Printf("close trace provider(%s) error(%s))", conf.Trace.EndpointUrl, err2)
		}
	}, nil
}
