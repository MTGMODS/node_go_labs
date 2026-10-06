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
	return newServerWithRepository(repository.NewInMemoryLicenseRepository())
}

func newServerWithRepository(repo repository.LicenseRepository) *httptest.Server {
	licenses := service.NewLicenseService(repo)
	return httptest.NewServer(router.New(controller.NewLicenseController(licenses)))
}

type failingRepository struct{}

func (failingRepository) List() ([]model.License, error) { return nil, errors.New("storage failed") }
func (failingRepository) FindByID(int) (model.License, error) {
	return model.License{}, errors.New("storage failed")
}
func (failingRepository) FindByKey(string) (model.License, error) {
	return model.License{}, errors.New("storage failed")
}
func (failingRepository) Create(model.License) (model.License, error) {
	return model.License{}, errors.New("storage failed")
}
func (failingRepository) Update(model.License) (model.License, error) {
	return model.License{}, errors.New("storage failed")
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

func licenseBody(fields map[string]any) io.Reader {
	body, _ := json.Marshal(fields)
	return bytes.NewReader(body)
}

func validLicense() map[string]any {
	return map[string]any{"key": "MTGM-VIP1-AAAA-0001", "product": "MTG MODS VIP", "owner": "student"}
}

func TestAPISuccessfulPostAndGet(t *testing.T) {
	server := newServer()
	defer server.Close()
	created := request(t, http.MethodPost, server.URL+"/licenses", licenseBody(validLicense()))
	defer created.Body.Close()
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("POST status=%d", created.StatusCode)
	}
	found := request(t, http.MethodGet, server.URL+"/licenses/1", nil)
	defer found.Body.Close()
	if found.StatusCode != http.StatusOK {
		t.Fatalf("GET status=%d", found.StatusCode)
	}
}

func TestAPIInvalidPost(t *testing.T) {
	server := newServer()
	defer server.Close()
	response := request(t, http.MethodPost, server.URL+"/licenses", licenseBody(map[string]any{"key": ""}))
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", response.StatusCode)
	}
}

func TestAPIMissingLicense(t *testing.T) {
	server := newServer()
	defer server.Close()
	response := request(t, http.MethodGet, server.URL+"/licenses/42", nil)
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", response.StatusCode)
	}
}

func TestAPIPartialUpdateAndDelete(t *testing.T) {
	server := newServer()
	defer server.Close()
	created := request(t, http.MethodPost, server.URL+"/licenses", licenseBody(validLicense()))
	created.Body.Close()
	updated := request(t, http.MethodPut, server.URL+"/licenses/1", licenseBody(map[string]any{"status": "ACTIVE"}))
	defer updated.Body.Close()
	if updated.StatusCode != http.StatusOK {
		t.Fatalf("PUT status=%d", updated.StatusCode)
	}
	deleted := request(t, http.MethodDelete, server.URL+"/licenses/1", nil)
	defer deleted.Body.Close()
	if deleted.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status=%d", deleted.StatusCode)
	}
}

func TestAPIConflictAndInvalidJSON(t *testing.T) {
	server := newServer()
	defer server.Close()
	created := request(t, http.MethodPost, server.URL+"/licenses", licenseBody(validLicense()))
	created.Body.Close()
	duplicate := validLicense()
	duplicate["owner"] = "other"
	conflict := request(t, http.MethodPost, server.URL+"/licenses", licenseBody(duplicate))
	defer conflict.Body.Close()
	if conflict.StatusCode != http.StatusConflict {
		t.Fatalf("conflict status=%d", conflict.StatusCode)
	}
	invalid := request(t, http.MethodPost, server.URL+"/licenses", bytes.NewBufferString("{"))
	defer invalid.Body.Close()
	if invalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid JSON status=%d", invalid.StatusCode)
	}
}

func TestAPIInternalError(t *testing.T) {
	server := newServerWithRepository(failingRepository{})
	defer server.Close()
	response := request(t, http.MethodGet, server.URL+"/licenses", nil)
	defer response.Body.Close()
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", response.StatusCode)
	}
}
