package service_test

import (
	"errors"
	"testing"

	"node-go-labs/lab4/go-service/internal/model"
	"node-go-labs/lab4/go-service/internal/repository"
	"node-go-labs/lab4/go-service/internal/service"
)

func newService() *service.UserService {
	return service.NewUserService(repository.NewInMemoryUserRepository())
}

func TestServiceCRUD(t *testing.T) {
	users := newService()
	created, err := users.CreateUser(model.User{Name: " John Doe ", Email: " JOHN@EXAMPLE.COM "})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != 1 || created.Name != "John Doe" || created.Email != "john@example.com" {
		t.Fatalf("unexpected created user: %+v", created)
	}

	found, err := users.GetUser(created.ID)
	if err != nil || found != created {
		t.Fatalf("unexpected found user: %+v, err=%v", found, err)
	}

	updated, err := users.UpdateUser(created.ID, model.User{Name: "Jane", Email: "jane@example.com"})
	if err != nil || updated.Name != "Jane" {
		t.Fatalf("unexpected updated user: %+v, err=%v", updated, err)
	}

	if err := users.DeleteUser(created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := users.GetUser(created.ID); domainCode(err) != "user_not_found" {
		t.Fatalf("expected user_not_found, got %v", err)
	}
}

func TestServiceRejectsDuplicateEmail(t *testing.T) {
	users := newService()
	_, _ = users.CreateUser(model.User{Name: "John", Email: "john@example.com"})
	_, err := users.CreateUser(model.User{Name: "Other", Email: "JOHN@example.com"})
	if domainCode(err) != "email_conflict" {
		t.Fatalf("expected email_conflict, got %v", err)
	}
}

func TestServiceReportsMissingUser(t *testing.T) {
	_, err := newService().GetUser(42)
	if domainCode(err) != "user_not_found" {
		t.Fatalf("expected user_not_found, got %v", err)
	}
}

func domainCode(err error) string {
	var domainError *service.DomainError
	if errors.As(err, &domainError) {
		return domainError.Code
	}
	return ""
}
