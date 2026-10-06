package service_test

import (
	"errors"
	"testing"

	"node-go-labs/lab4/go-service/internal/model"
	"node-go-labs/lab4/go-service/internal/repository"
	"node-go-labs/lab4/go-service/internal/service"
)

func stringPointer(value string) *string { return &value }
func intPointer(value int) *int          { return &value }

func newService() *service.LicenseService {
	return service.NewLicenseService(repository.NewInMemoryLicenseRepository())
}

func licenseInput() model.LicenseInput {
	return model.LicenseInput{
		Key: stringPointer("MTGM-VIP1-AAAA-0001"), Product: stringPointer("MTG MODS VIP"), Owner: stringPointer("student"),
	}
}

func TestServiceCRUDAndDefaults(t *testing.T) {
	licenses := newService()
	created, err := licenses.CreateLicense(licenseInput())
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != 1 || created.Status != "NOT_ACTIVATED" || created.DurationDays != 30 || created.MaxDevices != 1 {
		t.Fatalf("unexpected created license: %+v", created)
	}

	found, err := licenses.GetLicense(created.ID)
	if err != nil || found != created {
		t.Fatalf("unexpected found license: %+v, err=%v", found, err)
	}

	updated, err := licenses.UpdateLicense(created.ID, model.LicenseInput{Status: stringPointer("ACTIVE"), MaxDevices: intPointer(2)})
	if err != nil || updated.Key != created.Key || updated.Status != "ACTIVE" || updated.MaxDevices != 2 {
		t.Fatalf("unexpected updated license: %+v, err=%v", updated, err)
	}

	if err := licenses.DeleteLicense(created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := licenses.GetLicense(created.ID); domainCode(err) != "license_not_found" {
		t.Fatalf("expected license_not_found, got %v", err)
	}
}

func TestServiceRejectsDuplicateKey(t *testing.T) {
	licenses := newService()
	_, _ = licenses.CreateLicense(licenseInput())
	duplicate := licenseInput()
	duplicate.Owner = stringPointer("other")
	_, err := licenses.CreateLicense(duplicate)
	if domainCode(err) != "license_key_conflict" {
		t.Fatalf("expected license_key_conflict, got %v", err)
	}
}

func TestServiceReportsMissingLicense(t *testing.T) {
	_, err := newService().GetLicense(42)
	if domainCode(err) != "license_not_found" {
		t.Fatalf("expected license_not_found, got %v", err)
	}
}

func domainCode(err error) string {
	var domainError *service.DomainError
	if errors.As(err, &domainError) {
		return domainError.Code
	}
	return ""
}
