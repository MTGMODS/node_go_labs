package main

import (
	"context"
	"testing"
)

func TestPipelineProducesSameResultForDifferentConfigurations(t *testing.T) {
	first := runPipeline(context.Background(), Config{Tasks: 100, Workers: 1, Buffer: 0, Iterations: 100})
	second := runPipeline(context.Background(), Config{Tasks: 100, Workers: 8, Buffer: 100, Iterations: 100})

	if first.Completed != 100 || second.Completed != 100 {
		t.Fatalf("completed tasks: first=%d second=%d", first.Completed, second.Completed)
	}
	if first.Checksum != second.Checksum {
		t.Fatalf("checksums differ: first=%d second=%d", first.Checksum, second.Checksum)
	}
}

func TestConfigValidation(t *testing.T) {
	if _, err := parseConfig([]string{"-workers=3"}); err == nil {
		t.Fatal("expected invalid worker count to fail")
	}
	if _, err := parseConfig([]string{"-buffer=5"}); err == nil {
		t.Fatal("expected invalid buffer size to fail")
	}
	if _, err := parseConfig([]string{"-tasks=1000", "-workers=16", "-buffer=1000"}); err != nil {
		t.Fatalf("valid configuration failed: %v", err)
	}
}

func TestCanceledPipelineStopsCleanly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stats := runPipeline(ctx, Config{Tasks: 1000, Workers: 4, Buffer: 10, Iterations: 100})
	if !stats.Canceled {
		t.Fatal("expected pipeline to report cancellation")
	}
}
