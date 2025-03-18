package metrics

import (
	"testing"
)

// TestRandomSampler tests the RandomSampler
func TestRandomSampler(t *testing.T) {
	// Test with rate 1.0 (always sample)
	sampler := NewRandomSampler(1.0)
	if !sampler.Sample() {
		t.Error("Expected Sample() to return true with rate 1.0")
	}

	// Test with rate 0.0 (never sample)
	sampler = NewRandomSampler(0.0)
	if sampler.Sample() {
		t.Error("Expected Sample() to return false with rate 0.0")
	}
}

// TestTags tests the Tags map
func TestTags(t *testing.T) {
	// Create a new Tags map
	tags := Tags{
		"key1": "value1",
		"key2": "value2",
	}

	// Check that the map has the expected values
	if tags["key1"] != "value1" {
		t.Errorf("Expected tags[\"key1\"] to be \"value1\", got %q", tags["key1"])
	}
	if tags["key2"] != "value2" {
		t.Errorf("Expected tags[\"key2\"] to be \"value2\", got %q", tags["key2"])
	}

	// Check that a non-existent key returns the zero value
	if tags["key3"] != "" {
		t.Errorf("Expected tags[\"key3\"] to be \"\", got %q", tags["key3"])
	}
}

// TestMetricType tests the MetricType constants
func TestMetricType(t *testing.T) {
	// Check that the MetricType constants have the expected values
	if CounterType != "counter" {
		t.Errorf("Expected CounterType to be \"counter\", got %q", CounterType)
	}
	if GaugeType != "gauge" {
		t.Errorf("Expected GaugeType to be \"gauge\", got %q", GaugeType)
	}
	if HistogramType != "histogram" {
		t.Errorf("Expected HistogramType to be \"histogram\", got %q", HistogramType)
	}
	if SummaryType != "summary" {
		t.Errorf("Expected SummaryType to be \"summary\", got %q", SummaryType)
	}
}

// TestMetricsMiddlewareConfig tests the MetricsMiddlewareConfig struct
func TestMetricsMiddlewareConfig(t *testing.T) {
	// Create a new MetricsMiddlewareConfig
	config := MetricsMiddlewareConfig{
		EnableLatency:    true,
		EnableThroughput: true,
		EnableQPS:        true,
		EnableErrors:     true,
		SamplingRate:     0.5,
		DefaultTags: Tags{
			"key1": "value1",
			"key2": "value2",
		},
	}

	// Check that the config has the expected values
	if !config.EnableLatency {
		t.Error("Expected EnableLatency to be true")
	}
	if !config.EnableThroughput {
		t.Error("Expected EnableThroughput to be true")
	}
	if !config.EnableQPS {
		t.Error("Expected EnableQPS to be true")
	}
	if !config.EnableErrors {
		t.Error("Expected EnableErrors to be true")
	}
	if config.SamplingRate != 0.5 {
		t.Errorf("Expected SamplingRate to be 0.5, got %f", config.SamplingRate)
	}
	if config.DefaultTags["key1"] != "value1" {
		t.Errorf("Expected DefaultTags[\"key1\"] to be \"value1\", got %q", config.DefaultTags["key1"])
	}
	if config.DefaultTags["key2"] != "value2" {
		t.Errorf("Expected DefaultTags[\"key2\"] to be \"value2\", got %q", config.DefaultTags["key2"])
	}
}
