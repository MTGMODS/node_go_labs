package service

import (
	"errors"
	"fmt"
	"strings"

	"node-go-labs/lab4/go-service/internal/model"
	"node-go-labs/lab4/go-service/internal/repository"
)

type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string { return e.Message }

type UserService struct {
	repository repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repository: repo}
}

func (s *UserService) ListUsers() ([]model.User, error) {
	return s.repository.List()
}

func (s *UserService) GetUser(id int) (model.User, error) {
	user, err := s.repository.FindByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		return model.User{}, userNotFound(id)
	}
	return user, err
}

func (s *UserService) CreateUser(user model.User) (model.User, error) {
	user = normalize(user)
	if _, err := s.repository.FindByEmail(user.Email); err == nil {
		return model.User{}, emailConflict(user.Email)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return model.User{}, err
	}
	return s.repository.Create(user)
}

func (s *UserService) UpdateUser(id int, user model.User) (model.User, error) {
	if _, err := s.GetUser(id); err != nil {
		return model.User{}, err
	}
	user = normalize(user)
	owner, err := s.repository.FindByEmail(user.Email)
	if err == nil && owner.ID != id {
		return model.User{}, emailConflict(user.Email)
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return model.User{}, err
	}
	user.ID = id
	updated, err := s.repository.Update(user)
	if errors.Is(err, repository.ErrNotFound) {
		return model.User{}, userNotFound(id)
	}
	return updated, err
}

func (s *UserService) DeleteUser(id int) error {
	if _, err := s.GetUser(id); err != nil {
		return err
	}
	if err := s.repository.Delete(id); errors.Is(err, repository.ErrNotFound) {
		return userNotFound(id)
	} else {
		return err
	}
}

func normalize(user model.User) model.User {
	user.Name = strings.TrimSpace(user.Name)
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	return user
}

func userNotFound(id int) *DomainError {
	return &DomainError{Code: "user_not_found", Message: fmt.Sprintf("User with id %d was not found", id)}
}

func emailConflict(email string) *DomainError {
	return &DomainError{Code: "email_conflict", Message: fmt.Sprintf("User with email %s already exists", email)}
}
