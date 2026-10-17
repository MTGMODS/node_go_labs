package streaming

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const mebibyte = 1024 * 1024

var (
	ErrSourceNotFound = errors.New("stream source was not found")
	ErrJobNotFound    = errors.New("stream job was not found")
)

type EventPublisher func(value any)

type Job struct {
	ID               string  `json:"id"`
	Runtime          string  `json:"runtime"`
	Mode             string  `json:"mode"`
	Status           string  `json:"status"`
	SourceBytes      int64   `json:"sourceBytes"`
	ProcessedBytes   int64   `json:"processedBytes"`
	ProgressPercent  float64 `json:"progressPercent"`
	LineCount        int64   `json:"lineCount"`
	SlowDelayMS      int     `json:"slowDelayMs"`
	ChunkSizeKB      int     `json:"chunkSizeKB"`
	OutputFile       string  `json:"outputFile"`
	StartedAt        string  `json:"startedAt"`
	CompletedAt      string  `json:"completedAt,omitempty"`
	DurationMS       float64 `json:"durationMs,omitempty"`
	PeakAllocated    uint64  `json:"peakAllocatedBytes,omitempty"`
	AllocationGrowth uint64  `json:"allocationGrowthBytes,omitempty"`
	Error            string  `json:"error,omitempty"`
}

type SourceInfo struct {
	Exists    bool    `json:"exists"`
	Path      string  `json:"path"`
	SizeBytes int64   `json:"sizeBytes,omitempty"`
	SizeMB    float64 `json:"sizeMB,omitempty"`
}

type Service struct {
	dataDir    string
	sourcePath string
	publish    EventPublisher
	mu         sync.RWMutex
	jobs       map[string]*Job
}

func NewService(dataDir string, publish EventPublisher) *Service {
	if dataDir == "" {
		dataDir = os.Getenv("DATA_DIR")
	}
	if dataDir == "" {
		dataDir = "data"
	}
	if publish == nil {
		publish = func(any) {}
	}
	return &Service{
		dataDir:    dataDir,
		sourcePath: filepath.Join(dataDir, "licenses-source.ndjson"),
		publish:    publish,
		jobs:       make(map[string]*Job),
	}
}

func (s *Service) GenerateSource(sizeMB int) (SourceInfo, error) {
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return SourceInfo{}, err
	}
	file, err := os.Create(s.sourcePath)
	if err != nil {
		return SourceInfo{}, err
	}
	defer file.Close()

	record, _ := json.Marshal(map[string]string{
		"key": "MTGM-STREAM-DEMO", "product": "MTG MODS VIP",
		"owner": "stream-user", "status": "NOT_ACTIVATED",
	})
	record = append(record, '\n')
	block := bytes.Repeat(record, (64*1024/len(record))+1)
	target := int64(sizeMB) * mebibyte
	writer := bufio.NewWriterSize(file, 64*1024)
	for written := int64(0); written < target; {
		remaining := target - written
		chunk := block
		if int64(len(chunk)) > remaining {
			chunk = chunk[:remaining]
		}
		n, writeErr := writer.Write(chunk)
		if writeErr != nil {
			return SourceInfo{}, writeErr
		}
		written += int64(n)
	}
	if err := writer.Flush(); err != nil {
		return SourceInfo{}, err
	}
	return s.SourceInfo()
}

func (s *Service) SourceInfo() (SourceInfo, error) {
	stat, err := os.Stat(s.sourcePath)
	if errors.Is(err, os.ErrNotExist) {
		return SourceInfo{Exists: false, Path: filepath.Base(s.sourcePath)}, nil
	}
	if err != nil {
		return SourceInfo{}, err
	}
	return SourceInfo{
		Exists: true, Path: filepath.Base(s.sourcePath),
		SizeBytes: stat.Size(), SizeMB: float64(stat.Size()) / mebibyte,
	}, nil
}

func (s *Service) StartProcessing(mode string, slowDelayMS, chunkSizeKB int) (Job, error) {
	source, err := s.SourceInfo()
	if err != nil {
		return Job{}, err
	}
	if !source.Exists {
		return Job{}, ErrSourceNotFound
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	job := &Job{
		ID: id, Runtime: "go", Mode: mode, Status: "running",
		SourceBytes: source.SizeBytes, SlowDelayMS: slowDelayMS, ChunkSizeKB: chunkSizeKB,
		OutputFile: fmt.Sprintf("processed-%s-%s.ndjson", mode, id),
		StartedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}
	s.mu.Lock()
	s.jobs[id] = job
	s.mu.Unlock()
	go s.runJob(*job)
	return *job, nil
}

func (s *Service) GetJob(id string) (Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	return *job, nil
}

func (s *Service) ListJobs() []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jobs := make([]Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, *job)
	}
	return jobs
}

func (s *Service) WaitForJob(ctx context.Context, id string) (Job, error) {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		job, err := s.GetJob(id)
		if err != nil {
			return Job{}, err
		}
		if job.Status != "running" {
			return job, nil
		}
		select {
		case <-ctx.Done():
			return Job{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) runJob(job Job) {
	outputPath := filepath.Join(s.dataDir, job.OutputFile)
	started := time.Now()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	baseline := memory.Alloc
	peak := baseline

	progress := func(processed, lines int64) {
		runtime.ReadMemStats(&memory)
		if memory.Alloc > peak {
			peak = memory.Alloc
		}
		percent := float64(processed) / float64(job.SourceBytes) * 100
		s.updateJob(job.ID, func(current *Job) {
			current.ProcessedBytes = processed
			current.LineCount = lines
			current.ProgressPercent = percent
		})
		s.publish(map[string]any{
			"type": "progress", "jobId": job.ID, "runtime": "go", "mode": job.Mode,
			"processed": processed, "total": job.SourceBytes, "percent": percent,
		})
	}

	var err error
	if job.Mode == "safe" {
		err = processSafe(s.sourcePath, outputPath, job.ChunkSizeKB*1024, time.Duration(job.SlowDelayMS)*time.Millisecond, progress)
	} else {
		err = processUnsafe(s.sourcePath, outputPath, job.ChunkSizeKB*1024, time.Duration(job.SlowDelayMS)*time.Millisecond, progress)
	}

	s.updateJob(job.ID, func(current *Job) {
		current.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
		current.DurationMS = float64(time.Since(started).Microseconds()) / 1000
		current.PeakAllocated = peak
		if peak > baseline {
			current.AllocationGrowth = peak - baseline
		}
		if err != nil {
			current.Status = "failed"
			current.Error = err.Error()
		} else {
			current.Status = "completed"
			current.ProgressPercent = 100
		}
	})
	result, _ := s.GetJob(job.ID)
	if err != nil {
		s.publish(map[string]any{"type": "failed", "job": result})
	} else {
		s.publish(map[string]any{"type": "completed", "job": result})
	}
}

func (s *Service) updateJob(id string, update func(*Job)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job := s.jobs[id]; job != nil {
		update(job)
	}
}

func processSafe(sourcePath, outputPath string, chunkSize int, delay time.Duration, progress func(int64, int64)) error {
	input := make(chan []byte, 4)
	output := make(chan []byte, 4)
	errorsChannel := make(chan error, 2)

	go func() {
		defer close(input)
		errorsChannel <- produceChunks(sourcePath, chunkSize, func(chunk []byte) { input <- chunk })
	}()
	go func() {
		defer close(output)
		var processed, lines int64
		var lastByte byte
		hasData := false
		for chunk := range input {
			processed += int64(len(chunk))
			lines += int64(bytes.Count(chunk, []byte{'\n'}))
			lastByte = chunk[len(chunk)-1]
			hasData = true
			transformed := bytes.ToUpper(chunk)
			output <- transformed
			progress(processed, lines)
		}
		if hasData && lastByte != '\n' {
			lines++
			progress(processed, lines)
		}
		errorsChannel <- nil
	}()

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	writer := bufio.NewWriterSize(file, chunkSize)
	for chunk := range output {
		if delay > 0 {
			time.Sleep(delay)
		}
		if _, err = writer.Write(chunk); err != nil {
			file.Close()
			return err
		}
	}
	if err = writer.Flush(); err == nil {
		err = file.Close()
	} else {
		file.Close()
	}
	for range 2 {
		if pipelineErr := <-errorsChannel; pipelineErr != nil && err == nil {
			err = pipelineErr
		}
	}
	return err
}

func processUnsafe(sourcePath, outputPath string, chunkSize int, delay time.Duration, progress func(int64, int64)) error {
	queue := make([][]byte, 0)
	if err := produceChunks(sourcePath, chunkSize, func(chunk []byte) {
		queue = append(queue, chunk) // intentionally unbounded: demonstrates missing Backpressure
	}); err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriterSize(file, chunkSize)
	var processed, lines int64
	var lastByte byte
	hasData := false
	for _, chunk := range queue {
		if delay > 0 {
			time.Sleep(delay)
		}
		processed += int64(len(chunk))
		lines += int64(bytes.Count(chunk, []byte{'\n'}))
		lastByte = chunk[len(chunk)-1]
		hasData = true
		if _, err := writer.Write(bytes.ToUpper(chunk)); err != nil {
			return err
		}
		progress(processed, lines)
	}
	if hasData && lastByte != '\n' {
		lines++
		progress(processed, lines)
	}
	return writer.Flush()
}

func produceChunks(sourcePath string, chunkSize int, consume func([]byte)) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()
	for {
		buffer := make([]byte, chunkSize)
		n, readErr := file.Read(buffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			consume(chunk)
		}
		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func ValidMode(mode string) bool {
	return strings.EqualFold(mode, "safe") || strings.EqualFold(mode, "unsafe")
}
