package metrics

import (
	"testing"
	"time"
)

// TestPrometheusSummaryBuilder tests the PrometheusSummaryBuilder
func TestPrometheusSummaryBuilder(t *testing.T) {
	// Create a new registry
	registry := NewPrometheusRegistry()

	// Create a summary builder
	builder := registry.NewSummary()

	// Check that the builder is not nil
	if builder == nil {
		t.Fatal("Expected builder to not be nil")
	}

	// Set the name
	builder = builder.Name("test_summary")

	// Set the description
	builder = builder.Description("Test summary")

	// Add a tag
	builder = builder.Tag("key", "value")

	// Set the objectives
	builder = builder.Objectives(map[float64]float64{0.5: 0.05, 0.9: 0.01})

	// Set the max age
	builder = builder.MaxAge(10 * time.Second)

	// Set the age buckets
	builder = builder.AgeBuckets(5)

	// Build the summary
	summary := builder.Build()

	// Check that the summary is not nil
	if summary == nil {
		t.Fatal("Expected summary to not be nil")
	}

	// Check that the summary has the expected name
	if summary.Name() != "test_summary" {
		t.Errorf("Expected summary name to be \"test_summary\", got %q", summary.Name())
	}

	// Check that the summary has the expected description
	if summary.Description() != "Test summary" {
		t.Errorf("Expected summary description to be \"Test summary\", got %q", summary.Description())
	}

	// Check that the summary has the expected type
	if summary.Type() != SummaryType {
		t.Errorf("Expected summary type to be %q, got %q", SummaryType, summary.Type())
	}

	// Check that the summary has the expected tags
	tags := summary.Tags()
	if tags["key"] != "value" {
		t.Errorf("Expected summary tag \"key\" to be \"value\", got %q", tags["key"])
	}

	// Check that the summary has the expected objectives
	objectives := summary.Objectives()
	if len(objectives) != 2 {
		t.Errorf("Expected 2 objectives, got %d", len(objectives))
	}
	if objectives[0.5] != 0.05 {
		t.Errorf("Expected objective 0.5 to be 0.05, got %f", objectives[0.5])
	}
	if objectives[0.9] != 0.01 {
		t.Errorf("Expected objective 0.9 to be 0.01, got %f", objectives[0.9])
	}
}
