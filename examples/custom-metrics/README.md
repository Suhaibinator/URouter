# Custom Metrics Example

This example demonstrates how to use the enhanced metrics system (v2) in SRouter, including:

1. Using the built-in Prometheus metrics system
2. Implementing a custom metrics system
3. Using the metrics abstraction layer to support multiple metric formats

## Running the Example

### Prometheus Metrics (Default)

To run the example with Prometheus metrics:

```bash
go run .
```

This will start a server on port 8080 with the following endpoints:

- API: http://localhost:8080/api/users
- Metrics: http://localhost:8080/metrics

### Custom Metrics

To run the example with custom metrics:

```bash
go run . custom
```

This will start a server on port 8081 with the following endpoints:

- API: http://localhost:8081/api/custom
- Metrics: http://localhost:8081/metrics/custom

## Code Structure

- `main.go`: Main entry point that runs either the Prometheus or custom metrics example
- `custom_metrics.go`: Implementation of a custom metrics system
- `custom_example.go`: Example of using the custom metrics system

## Features Demonstrated

### Fluent API for Creating Metrics

```go
counter := registry.NewCounter().
    Name("http_requests_total").
    Description("Total number of HTTP requests").
    Tag("service", "api").
    Build()
```

### Tags Support

```go
counter.WithTags(v2.Tags{
    "method": "GET",
    "path":   "/api/users",
}).(v2.Counter).Inc()
```

### Separation of Collection and Exposition

```go
// Create a registry for collecting metrics
registry := v2.NewPrometheusRegistry()

// Create an exporter for exposing metrics
exporter := v2.NewPrometheusExporter(registry)

// Create a metrics handler
metricsHandler := exporter.Handler()
```

### Middleware for Automatic Metrics Collection

```go
middleware := v2.NewPrometheusMiddleware(registry, v2.MetricsMiddlewareConfig{
    EnableLatency:    true,
    EnableThroughput: true,
    EnableQPS:        true,
    EnableErrors:     true,
    DefaultTags: v2.Tags{
        "environment": "production",
        "version":     "1.0.0",
    },
})

// Use the middleware
mux.Handle("/", middleware.Handler("/", r))
```

### Custom Metrics Implementation

The `custom_metrics.go` file demonstrates how to implement a custom metrics system that conforms to the v2 metrics interfaces. This allows you to:

1. Use your own metrics collection system
2. Support multiple metric formats
3. Integrate with external monitoring systems

## Metrics Interfaces

The v2 metrics system is based on the following interfaces:

### MetricsRegistry

```go
type MetricsRegistry interface {
    // Register a metric with the registry
    Register(metric Metric) error
    
    // Get a metric by name
    Get(name string) (Metric, bool)
    
    // Unregister a metric from the registry
    Unregister(name string) bool
    
    // Clear all metrics from the registry
    Clear()
    
    // Get a snapshot of all metrics
    Snapshot() MetricsSnapshot
    
    // Create a new registry with the given tags
    WithTags(tags Tags) MetricsRegistry
    
    // Create a new counter builder
    NewCounter() CounterBuilder
    
    // Create a new gauge builder
    NewGauge() GaugeBuilder
    
    // Create a new histogram builder
    NewHistogram() HistogramBuilder
    
    // Create a new summary builder
    NewSummary() SummaryBuilder
}
```

### MetricsExporter

```go
type MetricsExporter interface {
    // Export metrics to the backend
    Export(snapshot MetricsSnapshot) error
    
    // Start the exporter
    Start() error
    
    // Stop the exporter
    Stop() error
    
    // Return an HTTP handler for exposing metrics
    Handler() http.Handler
}
```

### MetricsMiddleware

```go
type MetricsMiddleware interface {
    // Wrap an HTTP handler with metrics collection
    Handler(name string, handler http.Handler) http.Handler
    
    // Configure the middleware
    Configure(config MetricsMiddlewareConfig) MetricsMiddleware
    
    // Add a filter to the middleware
    WithFilter(filter MetricsFilter) MetricsMiddleware
    
    // Add a sampler to the middleware
    WithSampler(sampler MetricsSampler) MetricsMiddleware
}
```

## Benefits of the v2 Metrics System

1. **Fluent API**: More intuitive and chainable API for creating metrics
2. **Tags Support**: First-class support for metric tags (labels)
3. **Type Safety**: Strong typing for different metric types
4. **Separation of Concerns**: Clear separation between collection and exposition
5. **Extensibility**: Easy to extend with custom collectors and exporters
6. **Dependency Injection**: Support for injecting external metric collectors
7. **Multiple Formats**: Support for multiple metric formats
