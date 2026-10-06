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

type LicenseService struct {
	repository repository.LicenseRepository
}

func NewLicenseService(repo repository.LicenseRepository) *LicenseService {
	return &LicenseService{repository: repo}
}

func (s *LicenseService) ListLicenses() ([]model.License, error) {
	return s.repository.List()
}

func (s *LicenseService) GetLicense(id int) (model.License, error) {
	license, err := s.repository.FindByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		return model.License{}, licenseNotFound(id)
	}
	return license, err
}

func (s *LicenseService) CreateLicense(input model.LicenseInput) (model.License, error) {
	license := applyInput(model.License{
		Status:       "NOT_ACTIVATED",
		DurationDays: 30,
		MaxDevices:   1,
	}, input)
	if _, err := s.repository.FindByKey(license.Key); err == nil {
		return model.License{}, licenseKeyConflict(license.Key)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return model.License{}, err
	}
	return s.repository.Create(license)
}

func (s *LicenseService) UpdateLicense(id int, input model.LicenseInput) (model.License, error) {
	existing, err := s.GetLicense(id)
	if err != nil {
		return model.License{}, err
	}
	license := applyInput(existing, input)
	owner, err := s.repository.FindByKey(license.Key)
	if err == nil && owner.ID != id {
		return model.License{}, licenseKeyConflict(license.Key)
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return model.License{}, err
	}
	updated, err := s.repository.Update(license)
	if errors.Is(err, repository.ErrNotFound) {
		return model.License{}, licenseNotFound(id)
	}
	return updated, err
}

func (s *LicenseService) DeleteLicense(id int) error {
	if _, err := s.GetLicense(id); err != nil {
		return err
	}
	if err := s.repository.Delete(id); errors.Is(err, repository.ErrNotFound) {
		return licenseNotFound(id)
	} else {
		return err
	}
}

func applyInput(license model.License, input model.LicenseInput) model.License {
	if input.Key != nil {
		license.Key = strings.TrimSpace(*input.Key)
	}
	if input.Product != nil {
		license.Product = strings.TrimSpace(*input.Product)
	}
	if input.Owner != nil {
		license.Owner = strings.TrimSpace(*input.Owner)
	}
	if input.Status != nil {
		license.Status = *input.Status
	}
	if input.DurationDays != nil {
		license.DurationDays = *input.DurationDays
	}
	if input.MaxDevices != nil {
		license.MaxDevices = *input.MaxDevices
	}
	return license
}

func licenseNotFound(id int) *DomainError {
	return &DomainError{Code: "license_not_found", Message: fmt.Sprintf("License with id %d was not found", id)}
}

func licenseKeyConflict(key string) *DomainError {
	return &DomainError{Code: "license_key_conflict", Message: fmt.Sprintf("License with key %s already exists", key)}
}
