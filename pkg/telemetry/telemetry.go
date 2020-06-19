package telemetry

import (
	"net/http"
	"os"

	"github.com/Shopify/goose/logger"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	metricstdout "go.opentelemetry.io/otel/exporters/metric/stdout"
	tracerstdout "go.opentelemetry.io/otel/exporters/trace/stdout"
	"go.opentelemetry.io/otel/plugin/httptrace"
	"go.opentelemetry.io/otel/sdk/metric/controller/pull"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Providers

const STDOUT = "stdout"
const PRETTY = "pretty"
const PROMETHEUS = "prometheus"

var log = logger.New("telemetry")

type Cleanuper interface {
	Cleanup()
}

// traceMetricCleaner implements Cleanuper.
type traceMetricCleaner struct {
	tracer func()
	meter  func()
}

func (c *traceMetricCleaner) Cleanup() {
	c.tracer()
	c.meter()
}

func Initialize() Cleanuper {
	return &traceMetricCleaner{tracer: InitTracer(), meter: InitMeter()}
}

// InitTracer initializes the global trace provider.
func InitTracer() func() {
	// Some providers require cleanup
	cleanupFunc := func() {}

	tracerProvider := os.Getenv("TRACER_PROVIDER")
	if tracerProvider == "" {
		log(nil, nil).Info("TRACER_PROVIDER not set, tracing will not be generated.")
		return cleanupFunc
	}

	var exporter *tracerstdout.Exporter
	var err error
	switch tracerProvider {
	case STDOUT, PRETTY:
		exporter, err = tracerstdout.NewExporter(tracerstdout.Options{PrettyPrint: tracerProvider == PRETTY})
	default:
		log(nil, nil).WithField("provider", tracerProvider).Fatal("Unsupported trace provider")
	}

	if err != nil {
		log(nil, err).WithField("provider", tracerProvider).Fatal("failed to initialize exporter")
	}

	// For the demonstration, use sdktrace.AlwaysSample sampler to sample all traces.
	// In a production application, use sdktrace.ProbabilitySampler with a desired probability.
	tp, err := sdktrace.NewProvider(sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter))
	if err != nil {
		log(nil, err)
	}
	global.SetTraceProvider(tp)

	return cleanupFunc
}

// InitMeter initializes the global metric provider.
func InitMeter() func() {
	cleanupFunc := func() {}

	metricProvider := os.Getenv("METRIC_PROVIDER")
	if metricProvider == "" {
		log(nil, nil).Info("METRIC_PROVIDER not set, metrics will not be generated.")
		return cleanupFunc
	}

	var err error
	switch metricProvider {
	case STDOUT, PRETTY:
		var pusher *push.Controller
		pusher, err = metricstdout.InstallNewPipeline(metricstdout.Config{
			Quantiles:   []float64{},
			PrettyPrint: metricProvider == PRETTY,
		}, push.WithStateful(false))
		if err != nil {
			break
		}
		cleanupFunc = pusher.Stop
	case PROMETHEUS:
		var exporter *prometheus.Exporter
		exporter, err = prometheus.InstallNewPipeline(prometheus.Config{}, pull.WithStateful(false))
		if err != nil {
			break
		}
		http.HandleFunc("/metrics", exporter.ServeHTTP)
		go func() {
			_ = http.ListenAndServe(":2222", nil)
		}()
	default:
		log(nil, nil).WithField("provider", metricProvider).Fatal("Unsupported metric provider")
	}

	if err != nil {
		log(nil, err).WithField("provider", metricProvider).Fatal("failed to initialize metric stdout exporter")
	}

	initSystemStatsObserver()

	return cleanupFunc
}

// OpenTelemetryMiddleware adds monitoring around HTTP requests.
// Be careful not to add anything here that captures personally-identify information such as IP addresses.
func OpenTelemetryMiddleware(next http.Handler) http.Handler {
	tracer := global.Tracer("covidshield/request")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attrs, entries, spanCtx := httptrace.Extract(r.Context(), r)

		r = r.WithContext(correlation.ContextWithMap(r.Context(), correlation.NewMap(correlation.MapUpdate{
			MultiKV: entries,
		})))
		_, span := tracer.Start(
			trace.ContextWithRemoteSpanContext(r.Context(), spanCtx),
			"HTTP Request",
			trace.WithAttributes(attrs...),
		)
		defer span.End()
		next.ServeHTTP(w, r)
	})
}
