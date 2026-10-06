package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"node-go-labs/lab4/go-service/internal/httpjson"
	"node-go-labs/lab4/go-service/internal/model"
	"node-go-labs/lab4/go-service/internal/service"
)

var allowedStatuses = map[string]struct{}{
	"NOT_ACTIVATED": {},
	"ACTIVE":        {},
	"EXPIRED":       {},
	"BANNED":        {},
}

type LicenseController struct {
	service *service.LicenseService
}

func NewLicenseController(service *service.LicenseService) *LicenseController {
	return &LicenseController{service: service}
}

func (c *LicenseController) Health(w http.ResponseWriter, _ *http.Request) {
	httpjson.Write(w, http.StatusOK, map[string]string{
		"status": "UP", "service": "license-service", "runtime": "go",
	})
}

func (c *LicenseController) List(w http.ResponseWriter, _ *http.Request) {
	licenses, err := c.service.ListLicenses()
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, licenses)
}

func (c *LicenseController) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	license, err := c.service.GetLicense(id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, license)
}

func (c *LicenseController) Create(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeLicense(w, r, true)
	if !ok {
		return
	}
	created, err := c.service.CreateLicense(input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpjson.Write(w, http.StatusCreated, created)
}

func (c *LicenseController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	input, ok := decodeLicense(w, r, false)
	if !ok {
		return
	}
	updated, err := c.service.UpdateLicense(id, input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, updated)
}

func (c *LicenseController) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	if err := c.service.DeleteLicense(id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, raw string) (int, bool) {
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_identifier", "License id must be a positive integer")
		return 0, false
	}
	return id, true
}

func decodeLicense(w http.ResponseWriter, r *http.Request, creating bool) (model.LicenseInput, bool) {
	var input model.LicenseInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", "Request body contains invalid JSON")
		return model.LicenseInput{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", "Request body must contain one JSON object")
		return model.LicenseInput{}, false
	}
	if err := validateInput(input, creating); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return model.LicenseInput{}, false
	}
	return input, true
}

func validateInput(input model.LicenseInput, creating bool) error {
	if creating {
		if input.Key == nil {
			return errors.New("key is required")
		}
		if input.Product == nil {
			return errors.New("product is required")
		}
		if input.Owner == nil {
			return errors.New("owner is required")
		}
	} else if input.Key == nil && input.Product == nil && input.Owner == nil && input.Status == nil &&
		input.DurationDays == nil && input.MaxDevices == nil {
		return errors.New("at least one field is required")
	}

	for field, value := range map[string]*string{"key": input.Key, "product": input.Product, "owner": input.Owner} {
		if value != nil && strings.TrimSpace(*value) == "" {
			return fmt.Errorf("%s must be a non-empty string", field)
		}
	}
	if input.Status != nil {
		if _, ok := allowedStatuses[*input.Status]; !ok {
			return errors.New("status must be one of: NOT_ACTIVATED, ACTIVE, EXPIRED, BANNED")
		}
	}
	if input.DurationDays != nil && *input.DurationDays < 1 {
		return errors.New("duration_days must be an integer >= 1")
	}
	if input.MaxDevices != nil && *input.MaxDevices < 1 {
		return errors.New("max_devices must be an integer >= 1")
	}
	return nil
}

func writeServiceError(w http.ResponseWriter, err error) {
	var domainError *service.DomainError
	if errors.As(err, &domainError) {
		status := http.StatusBadRequest
		switch domainError.Code {
		case "license_not_found":
			status = http.StatusNotFound
		case "license_key_conflict":
			status = http.StatusConflict
		}
		httpjson.WriteError(w, status, domainError.Code, domainError.Message)
		return
	}
	httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
}
