package telemetry

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.11.0"
)

// // globalLabels defines the set of global labels that will be applied to all
// // metrics emitted using the telemetry package function wrappers.
// var globalLabels []metrics.Label
//
// // FormatPrometheus supported format type.
// const (
//
//	FormatPrometheus = "prometheus"
//
// )
const defaultMetricsCollectInterval = 10 * time.Second

// Config defines the configuration options for blobstream telemetry.
type Config struct {
	Metrics  bool   `mapstructure:"metrics" json:"metrics"`
	Endpoint string `mapstructure:"endpoint" json:"endpoint"`
	TLS      bool   `mapstructure:"tls" json:"tls"`
}

var (
	meter = otel.Meter("blobstream")
)

type Meters struct {
	ProcessedNonces   metric.Int64Counter
	FailedNonces      metric.Int64Counter
	ReprocessedNonces metric.Int64Counter
	ProcessingTime    metric.Float64Histogram
}

func InitMeters() (*Meters, error) {
	processedNonces, err := meter.Int64Counter("orchestrator_processed_nonces_counter",
		metric.WithDescription("the count of the nonces that have been successfuly processed by the orchestrator"))
	if err != nil {
		return nil, err
	}

	failedNonces, err := meter.Int64Counter("orchestrator_failed_nonces_counter",
		metric.WithDescription("the count of the nonces that couldn't be processed by the orchestrator"))
	if err != nil {
		return nil, err
	}

	reprocessedNonces, err := meter.Int64Counter("orchestrator_reprocessed_nonces_counter",
		metric.WithDescription("the count of the nonces that failed to be processed by the orchestrator but were requeued to be reprocessed subsequently"))
	if err != nil {
		return nil, err
	}

	processingTime, err := meter.Float64Histogram("orchestrator_processing_time",
		metric.WithDescription("the time it takes for a nonce to be processed or fail after it was picked up by the orchestrator processor"))
	if err != nil {
		return nil, err
	}
	return &Meters{
		ProcessedNonces:   processedNonces,
		FailedNonces:      failedNonces,
		ReprocessedNonces: reprocessedNonces,
		ProcessingTime:    processingTime,
	}, nil
}

func Start(
	ctx context.Context,
	logger tmlog.Logger,
	serviceName string,
	evmAddress common.Address,
	opts []otlpmetrichttp.Option,
) (*prometheus.Registry, func() error, error) {
	exp, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	provider := sdk.NewMeterProvider(
		sdk.WithReader(
			sdk.NewPeriodicReader(exp,
				sdk.WithTimeout(defaultMetricsCollectInterval),
				sdk.WithInterval(defaultMetricsCollectInterval))),
		sdk.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNamespaceKey.String("blobstream"),
				semconv.ServiceNameKey.String(serviceName),
				// ServiceInstanceIDKey will be exported with key: "instance"
				semconv.ServiceInstanceIDKey.String(evmAddress.Hex()),
			)))

	err = runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(defaultMetricsCollectInterval),
		runtime.WithMeterProvider(provider))
	if err != nil {
		return nil, nil, fmt.Errorf("start runtime metrics: %w", err)
	}

	prometheusRegistry := prometheus.NewRegistry()
	shutdown, err := PrometheusMetrics(ctx, logger, prometheusRegistry)

	return prometheusRegistry, shutdown, err
}

var (
	promAgentEndpoint = "/metrics"
	promAgentPort     = "8890"
)

// PrometheusMetrics option sets up native libp2p metrics up
func PrometheusMetrics(ctx context.Context, logger tmlog.Logger, registerer prometheus.Registerer) (func() error, error) {
	registry := registerer.(*prometheus.Registry)

	mux := http.NewServeMux()
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registerer})
	mux.Handle(promAgentEndpoint, handler)

	promHTTPServer := &http.Server{
		Addr:              fmt.Sprintf(":%s", promAgentPort),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	if err := promHTTPServer.ListenAndServe(); err != nil {
		logger.Error("Error starting Prometheus metrics exporter http server: %s", err)
	}

	logger.Info("Prometheus agent started on :%s/%s", promAgentPort, promAgentEndpoint)

	stopFunc := func() error {
		return promHTTPServer.Shutdown(ctx)
	}
	return stopFunc, nil
}

//// Metrics defines a wrapper around application telemetry functionality. It allows
//// metrics to be gathered at any point in time. When creating a Metrics object,
//// internally, a global metrics is registered with a set of sinks as configured
//// by the operator. In addition to the sinks, when a process gets a SIGUSR1, a
//// dump of formatted recent metrics will be sent to STDERR.
//type Metrics struct {
//	memSink           *metrics.InmemSink
//	prometheusEnabled bool
//}
//
//// GatherResponse is the response type of registered metrics
//type GatherResponse struct {
//	Metrics     []byte
//	ContentType string
//}
//
//// New creates a new instance of Metrics
//func New(cfg Config) (*Metrics, error) {
//	if !cfg.Enabled {
//		return nil, nil
//	}
//
//	if numGlobalLables := len(cfg.GlobalLabels); numGlobalLables > 0 {
//		parsedGlobalLabels := make([]metrics.Label, numGlobalLables)
//		for i, gl := range cfg.GlobalLabels {
//			parsedGlobalLabels[i] = NewLabel(gl[0], gl[1])
//		}
//
//		globalLabels = parsedGlobalLabels
//	}
//
//	metricsConf := metrics.DefaultConfig(cfg.ServiceName)
//	metricsConf.EnableHostname = cfg.EnableHostname
//	metricsConf.EnableHostnameLabel = cfg.EnableHostnameLabel
//
//	memSink := metrics.NewInmemSink(10*time.Second, time.Minute)
//	metrics.DefaultInmemSignal(memSink)
//
//	m := &Metrics{memSink: memSink}
//	fanout := metrics.FanoutSink{memSink}
//
//	if cfg.PrometheusRetentionTime > 0 {
//		m.prometheusEnabled = true
//		prometheusOpts := metricsprom.PrometheusOpts{
//			Expiration: time.Duration(cfg.PrometheusRetentionTime) * time.Second,
//		}
//
//		promSink, err := metricsprom.NewPrometheusSinkFrom(prometheusOpts)
//		if err != nil {
//			return nil, err
//		}
//
//		fanout = append(fanout, promSink)
//	}
//
//	if _, err := metrics.NewGlobal(metricsConf, fanout); err != nil {
//		return nil, err
//	}
//
//	return m, nil
//}
//
//// Gather collects all registered metrics and returns a GatherResponse where the
//// metrics are encoded depending on the type. Metrics are either encoded via
//// Prometheus or JSON if in-memory.
//func (m *Metrics) Gather(format string) (GatherResponse, error) {
//	switch format {
//	case FormatPrometheus:
//		return m.gatherPrometheus()
//
//	case FormatText:
//		return m.gatherGeneric()
//
//	case FormatDefault:
//		return m.gatherGeneric()
//
//	default:
//		return GatherResponse{}, fmt.Errorf("unsupported metrics format: %s", format)
//	}
//}
//
//func (m *Metrics) gatherPrometheus() (GatherResponse, error) {
//	if !m.prometheusEnabled {
//		return GatherResponse{}, fmt.Errorf("prometheus metrics are not enabled")
//	}
//
//	metricsFamilies, err := prometheus.DefaultGatherer.Gather()
//	if err != nil {
//		return GatherResponse{}, fmt.Errorf("failed to gather prometheus metrics: %w", err)
//	}
//
//	buf := &bytes.Buffer{}
//	defer buf.Reset()
//
//	e := expfmt.NewEncoder(buf, expfmt.FmtText)
//	for _, mf := range metricsFamilies {
//		if err := e.Encode(mf); err != nil {
//			return GatherResponse{}, fmt.Errorf("failed to encode prometheus metrics: %w", err)
//		}
//	}
//
//	return GatherResponse{ContentType: string(expfmt.FmtText), Metrics: buf.Bytes()}, nil
//}
//
//func (m *Metrics) gatherGeneric() (GatherResponse, error) {
//	summary, err := m.memSink.DisplayMetrics(nil, nil)
//	if err != nil {
//		return GatherResponse{}, fmt.Errorf("failed to gather in-memory metrics: %w", err)
//	}
//
//	content, err := json.Marshal(summary)
//	if err != nil {
//		return GatherResponse{}, fmt.Errorf("failed to encode in-memory metrics: %w", err)
//	}
//
//	return GatherResponse{ContentType: "application/json", Metrics: content}, nil
//}
