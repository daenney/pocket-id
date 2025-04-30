package bootstrap

import (
	"context"
	"log"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func defaultResource() *resource.Resource {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceName("pocket-id-backend"),
			semconv.ServiceVersion(common.Version),
		),
	)

	if err != nil {
		log.Fatalf("Failed to create resource: %s", err)
	}

	return r
}

func initMeter(ctx context.Context) metric.Reader {
	mr, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		log.Fatalf("Failed to create OTEL metric reader: %s", err)
	}
	return mr
}

func initTracer(ctx context.Context) sdktrace.SpanExporter {
	se, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		log.Fatalf("Failed to create OTEL span exporter: %s", err)
	}
	return se
}

func initOtel(ctx context.Context,
	metrics, traces bool,
) func() {
	shutdownFuncs := []func(){}
	if !traces {
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
		shutdownFuncs = append(shutdownFuncs, func() {})
	}
	if !metrics {
		otel.SetMeterProvider(metricnoop.NewMeterProvider())
		shutdownFuncs = append(shutdownFuncs, func() {})
	}

	resource := defaultResource()

	if traces {
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(resource),
			sdktrace.WithBatcher(initTracer(ctx)),
		)

		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(
			propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			),
		)

		shutdownFuncs = append(shutdownFuncs, func() {
			tpCtx, tpCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer tpCancel()
			if err := tp.Shutdown(tpCtx); err != nil {
				log.Printf("Failed to gracefully shut down traces exporter: %s", err)
			}
		})
	}

	if metrics {
		mp := metric.NewMeterProvider(
			metric.WithResource(resource),
			metric.WithReader(initMeter(ctx)),
		)

		otel.SetMeterProvider(mp)
		shutdownFuncs = append(shutdownFuncs, func() {
			mpCtx, mpCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer mpCancel()
			if err := mp.Shutdown(mpCtx); err != nil {
				log.Printf("Failed to gracefully shut down metrics exporter: %s", err)
			}
		})
	}

	return func() {
		for _, f := range shutdownFuncs {
			f()
		}
	}
}
