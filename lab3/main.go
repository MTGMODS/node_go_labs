package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/bits"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Job struct {
	ID    int
	Value uint64
}

type Result struct {
	JobID int
	Value uint64
}

type Config struct {
	Tasks      int
	Workers    int
	Buffer     int
	Iterations int
	Timeout    time.Duration
	Format     string
}

type Stats struct {
	Tasks              int     `json:"tasks"`
	Completed          int     `json:"completed"`
	Workers            int     `json:"workers"`
	Buffer             int     `json:"buffer"`
	Iterations         int     `json:"iterations_per_task"`
	DurationMS         float64 `json:"duration_ms"`
	ThroughputTasksSec float64 `json:"throughput_tasks_per_sec"`
	Checksum           uint64  `json:"checksum"`
	Canceled           bool    `json:"canceled"`
}

var validWorkers = map[int]bool{1: true, 2: true, 4: true, 8: true, 16: true}
var validBuffers = map[int]bool{0: true, 10: true, 100: true, 1000: true}

func parseConfig(args []string) (Config, error) {
	config := Config{}
	flags := flag.NewFlagSet("pipeline", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.IntVar(&config.Tasks, "tasks", 10_000, "number of jobs to produce")
	flags.IntVar(&config.Workers, "workers", 4, "worker count: 1, 2, 4, 8, or 16")
	flags.IntVar(&config.Buffer, "buffer", 100, "channel buffer: 0, 10, 100, or 1000")
	flags.IntVar(&config.Iterations, "iterations", 10_000, "calculation iterations per job")
	flags.DurationVar(&config.Timeout, "timeout", 0, "optional pipeline timeout, for example 500ms")
	flags.StringVar(&config.Format, "format", "text", "output format: text or json")

	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}
	if config.Tasks <= 0 {
		return Config{}, errors.New("tasks must be greater than zero")
	}
	if !validWorkers[config.Workers] {
		return Config{}, errors.New("workers must be one of: 1, 2, 4, 8, 16")
	}
	if !validBuffers[config.Buffer] {
		return Config{}, errors.New("buffer must be one of: 0, 10, 100, 1000")
	}
	if config.Iterations <= 0 {
		return Config{}, errors.New("iterations must be greater than zero")
	}
	if config.Timeout < 0 {
		return Config{}, errors.New("timeout cannot be negative")
	}
	if config.Format != "text" && config.Format != "json" {
		return Config{}, errors.New("format must be text or json")
	}
	return config, nil
}

func calculate(value uint64, iterations int) uint64 {
	result := value
	for index := 0; index < iterations; index++ {
		result = bits.RotateLeft64(result*6_364_136_223_846_793_005+1_442_695_040_888_963_407, 17)
	}
	return result
}

func producer(ctx context.Context, taskCount int, jobs chan<- Job) {
	defer close(jobs)
	for id := 1; id <= taskCount; id++ {
		job := Job{ID: id, Value: uint64(id)}
		select {
		case jobs <- job:
		case <-ctx.Done():
			return
		}
	}
}

func worker(ctx context.Context, jobs <-chan Job, results chan<- Result, iterations int) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			result := Result{JobID: job.ID, Value: calculate(job.Value, iterations)}
			select {
			case results <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

func runPipeline(ctx context.Context, config Config) Stats {
	startedAt := time.Now()
	jobs := make(chan Job, config.Buffer)
	results := make(chan Result, config.Buffer)

	var workerGroup sync.WaitGroup
	workerGroup.Add(config.Workers)
	for index := 0; index < config.Workers; index++ {
		go func() {
			defer workerGroup.Done()
			worker(ctx, jobs, results, config.Iterations)
		}()
	}

	go producer(ctx, config.Tasks, jobs)
	go func() {
		workerGroup.Wait()
		close(results)
	}()

	stats := Stats{
		Tasks:      config.Tasks,
		Workers:    config.Workers,
		Buffer:     config.Buffer,
		Iterations: config.Iterations,
	}

	for result := range results {
		stats.Completed++
		stats.Checksum += result.Value
	}

	stats.Canceled = ctx.Err() != nil
	stats.DurationMS = float64(time.Since(startedAt)) / float64(time.Millisecond)
	if stats.DurationMS > 0 {
		stats.ThroughputTasksSec = float64(stats.Completed) / (stats.DurationMS / 1000)
	}
	return stats
}

func printStats(stats Stats, format string) error {
	if format == "json" {
		encoder := json.NewEncoder(os.Stdout)
		return encoder.Encode(stats)
	}

	fmt.Println("Concurrent pipeline completed")
	fmt.Printf("Tasks: %d/%d\n", stats.Completed, stats.Tasks)
	fmt.Printf("Workers: %d\n", stats.Workers)
	fmt.Printf("Channel buffer: %d\n", stats.Buffer)
	fmt.Printf("Iterations per task: %d\n", stats.Iterations)
	fmt.Printf("Time: %.3f ms\n", stats.DurationMS)
	fmt.Printf("Throughput: %.2f tasks/s\n", stats.ThroughputTasksSec)
	fmt.Printf("Checksum: %d\n", stats.Checksum)
	fmt.Printf("Canceled: %t\n", stats.Canceled)
	return nil
}

func main() {
	config, err := parseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "configuration error:", err)
		os.Exit(2)
	}

	baseContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx := baseContext
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(baseContext, config.Timeout)
		defer cancel()
	}

	stats := runPipeline(ctx, config)
	if err := printStats(stats, config.Format); err != nil {
		fmt.Fprintln(os.Stderr, "output error:", err)
		os.Exit(1)
	}

	if stats.Canceled {
		fmt.Fprintf(os.Stderr, "pipeline stopped: %v\n", ctx.Err())
	}
}
