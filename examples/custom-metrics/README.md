# Custom Metrics Example

This example demonstrates how to implement custom metrics collection in SRouter, showcasing several advanced patterns:

1. **External Metric Collectors**: Shows how to allow external systems to collect metrics, rather than only supporting direct Prometheus integration.

2. **Dependency Injection for Metrics**: Demonstrates how to accept an external registry as a parameter instead of creating an internal one.

3. **Middleware for Custom Metrics**: Provides middleware options that allow for custom metric collection without modifying core functionality.

4. **Separation of Collection and Exposition**: Separates the collection of metrics from how they're exposed, allowing applications to decide how to expose metrics.

5. **Support for Multiple Metric Formats**: Shows how to support different metric formats beyond Prometheus.

## Running the Example

To run this example:

```bash
go run .
```

Then visit:
- http://localhost:8080/ to generate some metrics
- http://localhost:8080/metrics to view the collected metrics

## Key Components

### CustomMetricsRegistry

A simple metrics registry that allows for dependency injection of a Prometheus registry. This demonstrates how to accept an external registry rather than creating one internally.

### MetricsCollector Interface

An interface that abstracts the collection of metrics, allowing for different implementations. This demonstrates how to separate metrics collection from the core functionality.

### PrometheusMetricsCollector

An implementation of the MetricsCollector interface that uses Prometheus. This shows one way to collect metrics, but other implementations could be created for different metric systems.

### MetricsMiddleware

Middleware that uses a MetricsCollector to collect metrics for HTTP requests. This demonstrates how to provide middleware options for custom metric collection.

## Design Patterns

This example demonstrates several important design patterns for metrics:

1. **Dependency Injection**: The metrics registry and collector are injected into components that need them, rather than being created internally.

2. **Interface Abstraction**: The MetricsCollector interface abstracts the collection of metrics, allowing for different implementations.

3. **Middleware Pattern**: The MetricsMiddleware wraps HTTP handlers to collect metrics without modifying the core functionality.

4. **Separation of Concerns**: The collection of metrics is separated from how they're exposed, allowing applications to decide how to expose metrics.

These patterns make the metrics system more flexible and extensible, allowing for custom metric collection without modifying the core functionality.
