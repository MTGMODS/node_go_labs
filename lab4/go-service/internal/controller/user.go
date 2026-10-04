package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"node-go-labs/lab4/go-service/internal/httpjson"
	"node-go-labs/lab4/go-service/internal/model"
	"node-go-labs/lab4/go-service/internal/service"
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type UserController struct {
	service *service.UserService
}

func NewUserController(service *service.UserService) *UserController {
	return &UserController{service: service}
}

func (c *UserController) Health(w http.ResponseWriter, _ *http.Request) {
	httpjson.Write(w, http.StatusOK, map[string]string{"status": "UP", "runtime": "go"})
}

func (c *UserController) List(w http.ResponseWriter, _ *http.Request) {
	users, err := c.service.ListUsers()
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, users)
}

func (c *UserController) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	user, err := c.service.GetUser(id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, user)
}

func (c *UserController) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := decodeUser(w, r)
	if !ok {
		return
	}
	created, err := c.service.CreateUser(user)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpjson.Write(w, http.StatusCreated, created)
}

func (c *UserController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	user, ok := decodeUser(w, r)
	if !ok {
		return
	}
	updated, err := c.service.UpdateUser(id, user)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, updated)
}

func (c *UserController) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	if err := c.service.DeleteUser(id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, raw string) (int, bool) {
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_identifier", "User id must be a positive integer")
		return 0, false
	}
	return id, true
}

func decodeUser(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	var user model.User
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&user); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", "Request body contains invalid JSON")
		return model.User{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", "Request body must contain one JSON object")
		return model.User{}, false
	}
	if !validUser(user) {
		httpjson.WriteError(w, http.StatusBadRequest, "validation_error", "name and a valid email are required")
		return model.User{}, false
	}
	return user, true
}

func validUser(user model.User) bool {
	return strings.TrimSpace(user.Name) != "" && emailPattern.MatchString(strings.TrimSpace(user.Email))
}

func writeServiceError(w http.ResponseWriter, err error) {
	var domainError *service.DomainError
	if errors.As(err, &domainError) {
		status := http.StatusBadRequest
		switch domainError.Code {
		case "user_not_found":
			status = http.StatusNotFound
		case "email_conflict":
			status = http.StatusConflict
		}
		httpjson.WriteError(w, status, domainError.Code, domainError.Message)
		return
	}
	httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
}
