package streaming_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"node-go-labs/go-service/internal/model"
	"node-go-labs/go-service/internal/repository"
	"node-go-labs/go-service/internal/streaming"
)

func TestDatabaseExportStreamsRepositoryRecordsToCSV(t *testing.T) {
	directory := t.TempDir()
	repo := repository.NewInMemoryLicenseRepository()
	for _, key := range []string{"EXPORT-1", "EXPORT-2"} {
		_, err := repo.Create(model.License{Key: key, Product: "VIP", Owner: "student", Status: "ACTIVE", DurationDays: 30, MaxDevices: 1})
		if err != nil {
			t.Fatal(err)
		}
	}
	var progress int
	service := streaming.NewDatabaseExportService(repo, directory, func(value any) {
		if event, ok := value.(map[string]any); ok && event["type"] == "export_progress" {
			progress++
		}
	})
	result, err := service.Export(1)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(directory, result.Filename))
	if err != nil {
		t.Fatal(err)
	}
	if result.Processed != 2 || progress != 2 {
		t.Fatalf("processed=%d progress=%d", result.Processed, progress)
	}
	if !strings.Contains(string(content), "EXPORT-1") || !strings.Contains(string(content), "EXPORT-2") {
		t.Fatalf("missing exported license: %s", content)
	}
}
