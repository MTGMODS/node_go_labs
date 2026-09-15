package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

type Server struct {
	store *Store
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, errors.New("id must be a positive integer")
	}
	return id, nil
}

func readPayload(r *http.Request) (LicensePayload, error) {
	var payload LicensePayload
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		return LicensePayload{}, &validationError{msg: "invalid JSON"}
	}
	return payload, nil
}

func handleStoreError(w http.ResponseWriter, err error) {
	var vErr *validationError
	if errors.As(err, &vErr) {
		writeError(w, http.StatusBadRequest, vErr.Error())
		return
	}
	if errors.Is(err, errConflict) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}

type healthResponse struct {
	Status   string `json:"status"`
	Service  string `json:"service"`
	Runtime  string `json:"runtime"`
	Database string `json:"database"`
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	if err := s.store.Ping(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, healthResponse{
			Status:   "DOWN",
			Service:  "license-service",
			Runtime:  "go",
			Database: "DOWN",
		})
		return
	}
	writeJSON(w, http.StatusOK, healthResponse{
		Status:   "UP",
		Service:  "license-service",
		Runtime:  "go",
		Database: "UP",
	})
}

func methodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (s *Server) listLicenses(w http.ResponseWriter, _ *http.Request) {
	items, err := s.store.List()
	if err != nil {
		handleStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createLicense(w http.ResponseWriter, r *http.Request) {
	payload, err := readPayload(r)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	created, err := s.store.Create(payload)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) getLicense(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	license, err := s.store.Get(id)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, license)
}

func (s *Server) updateLicense(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := readPayload(r)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	updated, err := s.store.Update(id, payload)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteLicense(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.store.Delete(id); err != nil {
		handleStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /licenses", s.listLicenses)
	mux.HandleFunc("POST /licenses", s.createLicense)
	mux.HandleFunc("GET /licenses/{id}", s.getLicense)
	mux.HandleFunc("PUT /licenses/{id}", s.updateLicense)
	mux.HandleFunc("DELETE /licenses/{id}", s.deleteLicense)
	mux.HandleFunc("/health", methodNotAllowed)
	mux.HandleFunc("/licenses", methodNotAllowed)
	mux.HandleFunc("/licenses/{id}", methodNotAllowed)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)
		if pattern == "" {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		mux.ServeHTTP(w, r)
	})
}
