package config

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

func TracerProvider() (*tracesdk.TracerProvider, error) {
	client := otlptracehttp.NewClient()
	ctx := context.Background()
	exp, err := otlptrace.New(ctx, client)

	if err != nil {
		panic(err)
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("stateful-app"),
		)),
	)
	return tp, nil
}
