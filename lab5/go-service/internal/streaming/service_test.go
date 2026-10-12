package streaming_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"node-go-labs/lab5/go-service/internal/streaming"
)

func TestServiceGeneratesAndProcessesSource(t *testing.T) {
	dataDir := t.TempDir()
	events := make([]any, 0)
	var eventsMu sync.Mutex
	service := streaming.NewService(dataDir, func(event any) {
		eventsMu.Lock()
		events = append(events, event)
		eventsMu.Unlock()
	})

	source, err := service.GenerateSource(1)
	if err != nil {
		t.Fatal(err)
	}
	if source.SizeBytes != 1024*1024 {
		t.Fatalf("source size=%d", source.SizeBytes)
	}
	sourceData, err := os.ReadFile(filepath.Join(dataDir, "licenses-source.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	expectedOutput := bytes.ToUpper(sourceData)

	for _, mode := range []string{"safe", "unsafe"} {
		job, err := service.StartProcessing(mode, 0, 16)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		result, err := service.WaitForJob(ctx, job.ID)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != "completed" || result.ProcessedBytes != source.SizeBytes || result.LineCount == 0 {
			t.Fatalf("unexpected %s result: %+v", mode, result)
		}
		output, err := os.ReadFile(filepath.Join(dataDir, result.OutputFile))
		if err != nil {
			t.Fatal(err)
		}
		if len(output) != len(sourceData) || !bytes.Equal(output, expectedOutput) {
			t.Fatalf("%s output does not match transformed source", mode)
		}
	}
	eventsMu.Lock()
	eventCount := len(events)
	eventsMu.Unlock()
	if eventCount == 0 {
		t.Fatal("expected progress events")
	}
}

func TestServiceRequiresSource(t *testing.T) {
	service := streaming.NewService(t.TempDir(), nil)
	_, err := service.StartProcessing("safe", 0, 16)
	if !errors.Is(err, streaming.ErrSourceNotFound) {
		t.Fatalf("expected ErrSourceNotFound, got %v", err)
	}
}
