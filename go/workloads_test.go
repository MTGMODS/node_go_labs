package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCPUImplementationsPerformEqualWork(t *testing.T) {
	const tasks = 4
	const iterations = int64(25)
	want := uint64(tasks) * uint64(iterations)

	if got := runSequentialCPU(tasks, iterations); got != want {
		t.Fatalf("sequential result = %d, want %d", got, want)
	}
	if got := runParallelCPU(tasks, iterations); got != want {
		t.Fatalf("parallel result = %d, want %d", got, want)
	}
}

func TestIOWorkload(t *testing.T) {
	server := &Server{}
	request := httptest.NewRequest(http.MethodGet, "/io?delay_ms=1", nil)
	response := httptest.NewRecorder()

	server.ioWorkload(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestWorkloadValidation(t *testing.T) {
	server := &Server{}
	request := httptest.NewRequest(http.MethodGet, "/cpu?tasks=0", nil)
	response := httptest.NewRecorder()

	server.cpuSequential(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}
