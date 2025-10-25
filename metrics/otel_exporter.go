package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// OTelExporter provides OpenTelemetry metrics export following OTel standards
type OTelExporter struct {
	meterProvider *sdkmetric.MeterProvider
	collector     Collector

	// OTel meters and instruments
	meter                  metric.Meter
	queueLengthGauge      metric.Int64ObservableGauge
	statusCountGauge      metric.Int64ObservableGauge
	throughputGauge       metric.Int64ObservableGauge
	activeWorkersGauge    metric.Int64ObservableGauge
}

// NewOTelExporter creates a new OpenTelemetry metrics exporter with Prometheus format
func NewOTelExporter(collector Collector) (*OTelExporter, error) {
	// Create Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("creating prometheus exporter: %w", err)
	}

	// Create meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)
	otel.SetMeterProvider(meterProvider)

	// Create meter with service info
	meter := meterProvider.Meter(
		"webhook-inbox",
		metric.WithInstrumentationVersion("1.0.0"),
	)

	oe := &OTelExporter{
		meterProvider: meterProvider,
		collector:     collector,
		meter:         meter,
	}

	// Register metrics instruments
	if err := oe.registerInstruments(); err != nil {
		return nil, fmt.Errorf("registering instruments: %w", err)
	}

	return oe, nil
}

// registerInstruments creates and registers all OpenTelemetry metric instruments
func (oe *OTelExporter) registerInstruments() error {
	var err error

	// Queue length gauge (per route)
	oe.queueLengthGauge, err = oe.meter.Int64ObservableGauge(
		"webhook.queue.length",
		metric.WithDescription("Number of pending webhooks in the queue per route"),
		metric.WithUnit("{webhooks}"),
		metric.WithInt64Callback(oe.observeQueueLengths),
	)
	if err != nil {
		return fmt.Errorf("creating queue length gauge: %w", err)
	}

	// Status count gauge (per status)
	oe.statusCountGauge, err = oe.meter.Int64ObservableGauge(
		"webhook.status.count",
		metric.WithDescription("Number of webhooks by status"),
		metric.WithUnit("{webhooks}"),
		metric.WithInt64Callback(oe.observeStatusCounts),
	)
	if err != nil {
		return fmt.Errorf("creating status count gauge: %w", err)
	}

	// Throughput gauge (delivered webhooks over time windows)
	oe.throughputGauge, err = oe.meter.Int64ObservableGauge(
		"webhook.throughput",
		metric.WithDescription("Number of webhooks delivered over time window"),
		metric.WithUnit("{webhooks}"),
		metric.WithInt64Callback(oe.observeThroughput),
	)
	if err != nil {
		return fmt.Errorf("creating throughput gauge: %w", err)
	}

	// Active workers gauge (per route)
	oe.activeWorkersGauge, err = oe.meter.Int64ObservableGauge(
		"webhook.workers.active",
		metric.WithDescription("Number of active workers per route"),
		metric.WithUnit("{workers}"),
		metric.WithInt64Callback(oe.observeActiveWorkers),
	)
	if err != nil {
		return fmt.Errorf("creating active workers gauge: %w", err)
	}

	return nil
}

// observeQueueLengths is a callback that reports queue lengths
func (oe *OTelExporter) observeQueueLengths(ctx context.Context, observer metric.Int64Observer) error {
	queueLengths, err := oe.collector.GetQueueLengths(ctx)
	if err != nil {
		return err
	}

	for routeID, length := range queueLengths {
		observer.Observe(length, metric.WithAttributes(
			attribute.String("route.id", routeID),
		))
	}

	return nil
}

// observeStatusCounts is a callback that reports webhook counts by status
func (oe *OTelExporter) observeStatusCounts(ctx context.Context, observer metric.Int64Observer) error {
	statusCounts, err := oe.collector.GetStatusCounts(ctx)
	if err != nil {
		return err
	}

	for status, count := range statusCounts {
		observer.Observe(count, metric.WithAttributes(
			attribute.String("webhook.status", status),
		))
	}

	return nil
}

// observeThroughput is a callback that reports throughput metrics
func (oe *OTelExporter) observeThroughput(ctx context.Context, observer metric.Int64Observer) error {
	throughput, err := oe.collector.GetThroughput(ctx)
	if err != nil {
		return err
	}

	observer.Observe(throughput.LastMinute, metric.WithAttributes(
		attribute.String("time.window", "1m"),
	))
	observer.Observe(throughput.LastFiveMinutes, metric.WithAttributes(
		attribute.String("time.window", "5m"),
	))
	observer.Observe(throughput.LastFifteenMinutes, metric.WithAttributes(
		attribute.String("time.window", "15m"),
	))

	return nil
}

// observeActiveWorkers is a callback that reports active worker counts
func (oe *OTelExporter) observeActiveWorkers(ctx context.Context, observer metric.Int64Observer) error {
	workers, err := oe.collector.GetActiveWorkers(ctx)
	if err != nil {
		return err
	}

	for routeID, workersList := range workers {
		observer.Observe(int64(len(workersList)), metric.WithAttributes(
			attribute.String("route.id", routeID),
		))
	}

	return nil
}

// ServeHTTP serves Prometheus-formatted metrics on the given HTTP handler
func (oe *OTelExporter) ServeHTTP() http.Handler {
	return promhttp.Handler()
}

// Shutdown gracefully shuts down the meter provider
func (oe *OTelExporter) Shutdown(ctx context.Context) error {
	if oe.meterProvider != nil {
		return oe.meterProvider.Shutdown(ctx)
	}
	return nil
}
