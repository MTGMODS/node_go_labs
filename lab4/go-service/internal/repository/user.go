package repository

import (
	"errors"

	"node-go-labs/lab4/go-service/internal/model"
)

var ErrNotFound = errors.New("user not found")

type UserRepository interface {
	List() ([]model.User, error)
	FindByID(id int) (model.User, error)
	FindByEmail(email string) (model.User, error)
	Create(user model.User) (model.User, error)
	Update(user model.User) (model.User, error)
	Delete(id int) error
}
