package controller

import (
	"net/http"
	"strconv"

	"node-go-labs/go-service/internal/httpjson"
	"node-go-labs/go-service/internal/streaming"
)

type ExportController struct {
	service *streaming.DatabaseExportService
}

func NewExportController(exportService *streaming.DatabaseExportService) *ExportController {
	return &ExportController{service: exportService}
}

func (c *ExportController) Export(w http.ResponseWriter, r *http.Request) {
	batchSize := 100
	if raw := r.URL.Query().Get("batch_size"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 1000 {
			httpjson.WriteError(w, http.StatusBadRequest, "validation_error", "batch_size must be an integer between 1 and 1000")
			return
		}
		batchSize = parsed
	}
	result, err := c.service.Export(batchSize)
	if err != nil {
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}
	httpjson.Write(w, http.StatusOK, result)
}
