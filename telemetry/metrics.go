package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.11.0"
)

const (
	defaultMetricsCollectInterval = 10 * time.Second
	globalMetricsNamespace        = "blobstream"
)

// Config defines the configuration options for blobstream telemetry.
type Config struct {
	Metrics  bool   `mapstructure:"metrics" json:"metrics"`
	Endpoint string `mapstructure:"endpoint" json:"endpoint"`
	TLS      bool   `mapstructure:"tls" json:"tls"`
	P2P      string `mapstructure:"p2p" json:"p2p"`
}

var meter = otel.Meter(globalMetricsNamespace)

type OrchestratorMeters struct {
	ProcessedNonces   metric.Int64Counter
	FailedNonces      metric.Int64Counter
	ReprocessedNonces metric.Int64Counter
	ProcessingTime    metric.Float64Histogram
}

func InitOrchestratorMeters() (*OrchestratorMeters, error) {
	processedNonces, err := meter.Int64Counter("orchestrator_processed_nonces_counter",
		metric.WithDescription("the count of the nonces that have been successfully processed by the orchestrator"))
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

	return &OrchestratorMeters{
		ProcessedNonces:   processedNonces,
		FailedNonces:      failedNonces,
		ReprocessedNonces: reprocessedNonces,
		ProcessingTime:    processingTime,
	}, nil
}

type RelayerMeters struct {
	ProcessedNonces metric.Int64Counter
	Failures        metric.Int64Counter
	ProcessingTime  metric.Float64Histogram
}

func InitRelayerMeters() (*RelayerMeters, error) {
	processedNonces, err := meter.Int64Counter("relayer_processed_nonces_counter",
		metric.WithDescription("the count of the nonces that have been successfully processed by the relayer"))
	if err != nil {
		return nil, err
	}

	failedNonces, err := meter.Int64Counter("relayer_number_of_failures",
		metric.WithDescription("the number of failures the relayer failed to relay a nonce"))
	if err != nil {
		return nil, err
	}

	processingTime, err := meter.Float64Histogram("relayer_processing_time",
		metric.WithDescription("the time it takes for a nonce to be processed by the relayer"))
	if err != nil {
		return nil, err
	}

	return &RelayerMeters{
		ProcessedNonces: processedNonces,
		Failures:        failedNonces,
		ProcessingTime:  processingTime,
	}, nil
}

func Start(
	ctx context.Context,
	logger tmlog.Logger,
	serviceName string,
	instanceID string,
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
				semconv.ServiceNamespaceKey.String(globalMetricsNamespace),
				semconv.ServiceNameKey.String(serviceName),
				// ServiceInstanceIDKey will be exported with key: "instance"
				semconv.ServiceInstanceIDKey.String(instanceID),
			),
		),
	)

	otel.SetMeterProvider(provider)
	logger.Info("global meter setup", "namespace", globalMetricsNamespace, "service_name_key", serviceName, "service_instance_id_key", instanceID)

	err = runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(defaultMetricsCollectInterval),
		runtime.WithMeterProvider(provider))
	if err != nil {
		return nil, nil, fmt.Errorf("start runtime metrics: %w", err)
	}

	prometheusRegistry := prometheus.NewRegistry()

	return prometheusRegistry, func() error {
		return provider.Shutdown(ctx)
	}, err
}

var promAgentEndpoint = "/metrics"

// PrometheusMetrics sets up native libp2p metrics up
func PrometheusMetrics(ctx context.Context, logger tmlog.Logger, registerer prometheus.Registerer, listenAddress string) (func() error, error) {
	registry := registerer.(*prometheus.Registry)

	mux := http.NewServeMux()
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registerer})
	mux.Handle(promAgentEndpoint, handler)

	promHTTPServer := &http.Server{
		Addr:              listenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := promHTTPServer.ListenAndServe(); err != nil {
			logger.Error("Error starting Prometheus metrics exporter http server: %s", err)
		}
	}()
	logger.Info(
		"Prometheus agent started",
		"listen_addr",
		fmt.Sprintf("%s%s", listenAddress, promAgentEndpoint),
	)

	stopFunc := func() error {
		return promHTTPServer.Shutdown(ctx)
	}
	return stopFunc, nil
}
