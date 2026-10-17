package repository

import (
	"errors"

	"node-go-labs/go-service/internal/model"
)

var ErrNotFound = errors.New("license not found")
var ErrConflict = errors.New("license key already exists")

type LicenseRepository interface {
	List() ([]model.License, error)
	FindByID(id int) (model.License, error)
	FindByKey(key string) (model.License, error)
	Create(license model.License) (model.License, error)
	Update(license model.License) (model.License, error)
	Delete(id int) error
}

type LicenseBatchReader interface {
	Count() (int, error)
	ListBatch(afterID, limit int) ([]model.License, error)
}
