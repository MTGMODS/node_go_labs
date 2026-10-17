package streaming

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"node-go-labs/go-service/internal/repository"
)

type ExportResult struct {
	Filename     string `json:"filename"`
	Processed    int    `json:"processed"`
	Total        int    `json:"total"`
	Backpressure string `json:"backpressure"`
}

type DatabaseExportService struct {
	repository repository.LicenseBatchReader
	dataDir    string
	publish    EventPublisher
}

func NewDatabaseExportService(repo repository.LicenseBatchReader, dataDir string, publish EventPublisher) *DatabaseExportService {
	if dataDir == "" {
		dataDir = "data"
	}
	if publish == nil {
		publish = func(any) {}
	}
	return &DatabaseExportService{repository: repo, dataDir: dataDir, publish: publish}
}

func (s *DatabaseExportService) Export(batchSize int) (ExportResult, error) {
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return ExportResult{}, err
	}
	total, err := s.repository.Count()
	if err != nil {
		return ExportResult{}, err
	}
	filename := fmt.Sprintf("licenses-go-%d.csv", time.Now().UnixMilli())
	output, err := os.Create(filepath.Join(s.dataDir, filename))
	if err != nil {
		return ExportResult{}, err
	}
	defer output.Close()

	reader, writer := io.Pipe()
	processedChannel := make(chan int, 1)
	go s.produceCSV(writer, batchSize, total, processedChannel)
	_, copyErr := io.CopyBuffer(output, reader, make([]byte, 64*1024))
	if copyErr != nil {
		return ExportResult{}, copyErr
	}
	processed := <-processedChannel
	s.publish(map[string]any{
		"type": "export_completed", "runtime": "go", "processed": processed,
		"total": total, "filename": filename, "timestamp": time.Now().UTC(),
	})
	return ExportResult{Filename: filename, Processed: processed, Total: total, Backpressure: "io.Pipe blocks producer until consumer reads"}, nil
}

func (s *DatabaseExportService) produceCSV(pipe *io.PipeWriter, batchSize, total int, completed chan<- int) {
	writer := csv.NewWriter(pipe)
	if err := writer.Write([]string{"id", "key", "product", "owner", "status", "duration_days", "max_devices"}); err != nil {
		pipe.CloseWithError(err)
		return
	}
	processed, afterID := 0, 0
	for {
		batch, err := s.repository.ListBatch(afterID, batchSize)
		if err != nil {
			pipe.CloseWithError(err)
			return
		}
		if len(batch) == 0 {
			break
		}
		for _, license := range batch {
			if err := writer.Write([]string{
				strconv.Itoa(license.ID), license.Key, license.Product, license.Owner, license.Status,
				strconv.Itoa(license.DurationDays), strconv.Itoa(license.MaxDevices),
			}); err != nil {
				pipe.CloseWithError(err)
				return
			}
			processed++
			afterID = license.ID
			s.publish(map[string]any{
				"type": "export_progress", "runtime": "go", "processed": processed,
				"total": total, "timestamp": time.Now().UTC(),
			})
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			pipe.CloseWithError(err)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		pipe.CloseWithError(err)
		return
	}
	completed <- processed
	close(completed)
	_ = pipe.Close()
}
