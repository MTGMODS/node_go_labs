package controller

import (
	"net/http"
	"strconv"

	"node-go-labs/go-service/internal/events"
	"node-go-labs/go-service/internal/httpjson"
)

type EventController struct {
	logger *events.MongoLogger
}

func NewEventController(logger *events.MongoLogger) *EventController {
	return &EventController{logger: logger}
}

func (c *EventController) List(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			httpjson.WriteError(w, http.StatusBadRequest, "validation_error", "limit must be an integer between 1 and 200")
			return
		}
		limit = parsed
	}
	eventList, err := c.logger.List(limit)
	if err != nil {
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}
	httpjson.Write(w, http.StatusOK, eventList)
}
