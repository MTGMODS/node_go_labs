package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"node-go-labs/go-service/internal/model"
	"node-go-labs/go-service/internal/repository"
)

type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string { return e.Message }

type LicenseService struct {
	repository repository.LicenseRepository
	publish    EventPublisher
}

type Event struct {
	Type      string        `json:"type" bson:"type"`
	Data      model.License `json:"data" bson:"data"`
	Timestamp time.Time     `json:"timestamp" bson:"timestamp"`
	Runtime   string        `json:"runtime" bson:"runtime"`
}

type EventPublisher func(Event) error

func NewLicenseService(repo repository.LicenseRepository, publishers ...EventPublisher) *LicenseService {
	publish := func(Event) error { return nil }
	if len(publishers) > 0 && publishers[0] != nil {
		publish = publishers[0]
	}
	return &LicenseService{repository: repo, publish: publish}
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
	created, err := s.repository.Create(license)
	if errors.Is(err, repository.ErrConflict) {
		return model.License{}, licenseKeyConflict(license.Key)
	}
	if err != nil {
		return model.License{}, err
	}
	if err := s.emit("license_created", created); err != nil {
		return model.License{}, err
	}
	return created, nil
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
	if errors.Is(err, repository.ErrConflict) {
		return model.License{}, licenseKeyConflict(license.Key)
	}
	if err != nil {
		return model.License{}, err
	}
	if err := s.emit("license_updated", updated); err != nil {
		return model.License{}, err
	}
	return updated, nil
}

func (s *LicenseService) DeleteLicense(id int) error {
	existing, err := s.GetLicense(id)
	if err != nil {
		return err
	}
	if err := s.repository.Delete(id); errors.Is(err, repository.ErrNotFound) {
		return licenseNotFound(id)
	} else if err != nil {
		return err
	}
	return s.emit("license_deleted", existing)
}

func (s *LicenseService) emit(eventType string, license model.License) error {
	return s.publish(Event{Type: eventType, Data: license, Timestamp: time.Now().UTC(), Runtime: "go"})
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
