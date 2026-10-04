package router_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"node-go-labs/lab4/go-service/internal/controller"
	"node-go-labs/lab4/go-service/internal/model"
	"node-go-labs/lab4/go-service/internal/repository"
	"node-go-labs/lab4/go-service/internal/router"
	"node-go-labs/lab4/go-service/internal/service"
)

func newServer() *httptest.Server {
	return newServerWithRepository(repository.NewInMemoryUserRepository())
}

func newServerWithRepository(repo repository.UserRepository) *httptest.Server {
	users := service.NewUserService(repo)
	return httptest.NewServer(router.New(controller.NewUserController(users)))
}

type failingRepository struct{}

func (failingRepository) List() ([]model.User, error) { return nil, errors.New("storage failed") }
func (failingRepository) FindByID(int) (model.User, error) {
	return model.User{}, errors.New("storage failed")
}
func (failingRepository) FindByEmail(string) (model.User, error) {
	return model.User{}, errors.New("storage failed")
}
func (failingRepository) Create(model.User) (model.User, error) {
	return model.User{}, errors.New("storage failed")
}
func (failingRepository) Update(model.User) (model.User, error) {
	return model.User{}, errors.New("storage failed")
}
func (failingRepository) Delete(int) error { return errors.New("storage failed") }

func request(t *testing.T, method, url string, body io.Reader) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func userBody(name, email string) io.Reader {
	body, _ := json.Marshal(map[string]string{"name": name, "email": email})
	return bytes.NewReader(body)
}

func TestAPISuccessfulPostAndGet(t *testing.T) {
	server := newServer()
	defer server.Close()
	created := request(t, http.MethodPost, server.URL+"/api/users", userBody("John", "john@example.com"))
	defer created.Body.Close()
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("POST status=%d", created.StatusCode)
	}
	found := request(t, http.MethodGet, server.URL+"/api/users/1", nil)
	defer found.Body.Close()
	if found.StatusCode != http.StatusOK {
		t.Fatalf("GET status=%d", found.StatusCode)
	}
}

func TestAPIInvalidPost(t *testing.T) {
	server := newServer()
	defer server.Close()
	response := request(t, http.MethodPost, server.URL+"/api/users", userBody("", "invalid"))
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", response.StatusCode)
	}
}

func TestAPIMissingUser(t *testing.T) {
	server := newServer()
	defer server.Close()
	response := request(t, http.MethodGet, server.URL+"/api/users/42", nil)
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", response.StatusCode)
	}
}

func TestAPIDelete(t *testing.T) {
	server := newServer()
	defer server.Close()
	created := request(t, http.MethodPost, server.URL+"/api/users", userBody("John", "john@example.com"))
	created.Body.Close()
	deleted := request(t, http.MethodDelete, server.URL+"/api/users/1", nil)
	defer deleted.Body.Close()
	if deleted.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", deleted.StatusCode)
	}
}

func TestAPIConflictAndInvalidJSON(t *testing.T) {
	server := newServer()
	defer server.Close()
	created := request(t, http.MethodPost, server.URL+"/api/users", userBody("John", "john@example.com"))
	created.Body.Close()
	conflict := request(t, http.MethodPost, server.URL+"/api/users", userBody("Other", "john@example.com"))
	defer conflict.Body.Close()
	if conflict.StatusCode != http.StatusConflict {
		t.Fatalf("conflict status=%d", conflict.StatusCode)
	}
	invalid := request(t, http.MethodPost, server.URL+"/api/users", bytes.NewBufferString("{"))
	defer invalid.Body.Close()
	if invalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid JSON status=%d", invalid.StatusCode)
	}
}

func TestAPIInternalError(t *testing.T) {
	server := newServerWithRepository(failingRepository{})
	defer server.Close()
	response := request(t, http.MethodGet, server.URL+"/api/users", nil)
	defer response.Body.Close()
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", response.StatusCode)
	}
}
