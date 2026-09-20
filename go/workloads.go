package main

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

const (
	defaultIODelayMS     int64 = 1000
	defaultCPUIterations int64 = 75_000_000
	defaultCPUTasks      int64 = 4
)

type workloadResponse struct {
	Status               string  `json:"status"`
	Workload             string  `json:"workload"`
	Runtime              string  `json:"runtime"`
	Operation            string  `json:"operation,omitempty"`
	Mode                 string  `json:"mode,omitempty"`
	DelayMS              int64   `json:"delay_ms,omitempty"`
	Tasks                int     `json:"tasks,omitempty"`
	IterationsPerTask    int64   `json:"iterations_per_task,omitempty"`
	TotalIterations      int64   `json:"total_iterations,omitempty"`
	Result               uint64  `json:"result,omitempty"`
	GOMAXPROCS           int     `json:"gomaxprocs,omitempty"`
	AvailableParallelism int     `json:"available_parallelism,omitempty"`
	DurationMS           float64 `json:"duration_ms"`
}

func queryInt64(r *http.Request, name string, fallback, min, max int64) (int64, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < min || value > max {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", name, min, max)
	}
	return value, nil
}

func increment(iterations int64) uint64 {
	var value uint64
	for index := int64(0); index < iterations; index++ {
		value++
	}
	return value
}

func runSequentialCPU(tasks int, iterations int64) uint64 {
	var result uint64
	for task := 0; task < tasks; task++ {
		result += increment(iterations)
	}
	return result
}

func runParallelCPU(tasks int, iterations int64) uint64 {
	results := make(chan uint64, tasks)
	for task := 0; task < tasks; task++ {
		go func() {
			results <- increment(iterations)
		}()
	}

	var result uint64
	for task := 0; task < tasks; task++ {
		result += <-results
	}
	return result
}

func (s *Server) ioWorkload(w http.ResponseWriter, r *http.Request) {
	delayMS, err := queryInt64(r, "delay_ms", defaultIODelayMS, 1, 30_000)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	startedAt := time.Now()
	timer := time.NewTimer(time.Duration(delayMS) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-timer.C:
		writeJSON(w, http.StatusOK, workloadResponse{
			Status:     "completed",
			Workload:   "io",
			Runtime:    "go",
			Operation:  "non-blocking time.Timer",
			DelayMS:    delayMS,
			DurationMS: float64(time.Since(startedAt)) / float64(time.Millisecond),
		})
	case <-r.Context().Done():
		return
	}
}

func (s *Server) cpuSequential(w http.ResponseWriter, r *http.Request) {
	s.cpuWorkload(w, r, "sequential")
}

func (s *Server) cpuParallel(w http.ResponseWriter, r *http.Request) {
	s.cpuWorkload(w, r, "parallel-goroutines")
}

func (s *Server) cpuWorkload(w http.ResponseWriter, r *http.Request, mode string) {
	iterations, err := queryInt64(r, "iterations", defaultCPUIterations, 1, 500_000_000)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tasksValue, err := queryInt64(r, "tasks", defaultCPUTasks, 1, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tasks := int(tasksValue)

	startedAt := time.Now()
	var result uint64
	if mode == "parallel-goroutines" {
		result = runParallelCPU(tasks, iterations)
	} else {
		result = runSequentialCPU(tasks, iterations)
	}

	writeJSON(w, http.StatusOK, workloadResponse{
		Status:               "completed",
		Workload:             "cpu",
		Runtime:              "go",
		Mode:                 mode,
		Tasks:                tasks,
		IterationsPerTask:    iterations,
		TotalIterations:      int64(tasks) * iterations,
		Result:               result,
		GOMAXPROCS:           runtime.GOMAXPROCS(0),
		AvailableParallelism: runtime.NumCPU(),
		DurationMS:           float64(time.Since(startedAt)) / float64(time.Millisecond),
	})
}
