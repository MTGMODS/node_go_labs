package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"node-go-labs/lab5/go-service/internal/httpjson"
	"node-go-labs/lab5/go-service/internal/streaming"
)

type StreamController struct {
	service *streaming.Service
}

func NewStreamController(service *streaming.Service) *StreamController {
	return &StreamController{service: service}
}

func (c *StreamController) SourceInfo(w http.ResponseWriter, _ *http.Request) {
	info, err := c.service.SourceInfo()
	if err != nil {
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}
	httpjson.Write(w, http.StatusOK, info)
}

func (c *StreamController) GenerateSource(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SizeMB int `json:"size_mb"`
	}
	if !decodeRequest(w, r, &request) {
		return
	}
	if request.SizeMB == 0 {
		request.SizeMB = 100
	}
	if request.SizeMB < 1 || request.SizeMB > 1024 {
		httpjson.WriteError(w, http.StatusBadRequest, "validation_error", "size_mb must be between 1 and 1024")
		return
	}
	info, err := c.service.GenerateSource(request.SizeMB)
	if err != nil {
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}
	httpjson.Write(w, http.StatusCreated, info)
}

func (c *StreamController) Start(w http.ResponseWriter, r *http.Request) {
	request := struct {
		Mode        string `json:"mode"`
		SlowDelayMS int    `json:"slow_delay_ms"`
		ChunkSizeKB int    `json:"chunk_size_kb"`
	}{Mode: "safe", SlowDelayMS: 2, ChunkSizeKB: 64}
	if !decodeRequest(w, r, &request) {
		return
	}
	if !streaming.ValidMode(request.Mode) {
		httpjson.WriteError(w, http.StatusBadRequest, "validation_error", "mode must be safe or unsafe")
		return
	}
	if request.SlowDelayMS < 0 || request.SlowDelayMS > 100 {
		httpjson.WriteError(w, http.StatusBadRequest, "validation_error", "slow_delay_ms must be between 0 and 100")
		return
	}
	if request.ChunkSizeKB < 4 || request.ChunkSizeKB > 1024 {
		httpjson.WriteError(w, http.StatusBadRequest, "validation_error", "chunk_size_kb must be between 4 and 1024")
		return
	}
	job, err := c.service.StartProcessing(request.Mode, request.SlowDelayMS, request.ChunkSizeKB)
	if errors.Is(err, streaming.ErrSourceNotFound) {
		httpjson.WriteError(w, http.StatusNotFound, "stream_source_not_found", "Generate the source file before starting processing")
		return
	}
	if err != nil {
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}
	httpjson.Write(w, http.StatusAccepted, job)
}

func (c *StreamController) ListJobs(w http.ResponseWriter, _ *http.Request) {
	httpjson.Write(w, http.StatusOK, c.service.ListJobs())
}

func (c *StreamController) GetJob(w http.ResponseWriter, r *http.Request) {
	job, err := c.service.GetJob(r.PathValue("id"))
	if errors.Is(err, streaming.ErrJobNotFound) {
		httpjson.WriteError(w, http.StatusNotFound, "stream_job_not_found", "Stream job was not found")
		return
	}
	httpjson.Write(w, http.StatusOK, job)
}

func decodeRequest(w http.ResponseWriter, r *http.Request, destination any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", "Request body contains invalid JSON")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", "Request body must contain one JSON object")
		return false
	}
	return true
}
