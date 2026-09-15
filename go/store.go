package main

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/lib/pq"
)

var allowedStatuses = map[string]struct{}{
	"NOT_ACTIVATED": {},
	"ACTIVE":        {},
	"EXPIRED":       {},
	"BANNED":        {},
}

var (
	errNotFound = errors.New("license not found")
	errConflict = errors.New("license key already exists")
)

type validationError struct {
	msg string
}

func (e *validationError) Error() string {
	return e.msg
}

type License struct {
	ID           int    `json:"id"`
	Key          string `json:"key"`
	Product      string `json:"product"`
	Owner        string `json:"owner"`
	Status       string `json:"status"`
	DurationDays int    `json:"duration_days"`
	MaxDevices   int    `json:"max_devices"`
}

type LicensePayload struct {
	Key          *string `json:"key"`
	Product      *string `json:"product"`
	Owner        *string `json:"owner"`
	Status       *string `json:"status"`
	DurationDays *int    `json:"duration_days"`
	MaxDevices   *int    `json:"max_devices"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ping() error {
	return s.db.Ping()
}

const licenseColumns = "id, key, product, owner, status, duration_days, max_devices"

func scanLicense(scanner interface{ Scan(dest ...any) error }) (License, error) {
	var license License
	err := scanner.Scan(
		&license.ID,
		&license.Key,
		&license.Product,
		&license.Owner,
		&license.Status,
		&license.DurationDays,
		&license.MaxDevices,
	)
	return license, err
}

func (s *Store) List() ([]License, error) {
	rows, err := s.db.Query("SELECT " + licenseColumns + " FROM licenses ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]License, 0)
	for rows.Next() {
		license, err := scanLicense(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, license)
	}
	return items, rows.Err()
}

func (s *Store) Get(id int) (License, error) {
	license, err := scanLicense(s.db.QueryRow(
		"SELECT "+licenseColumns+" FROM licenses WHERE id = $1",
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return License{}, errNotFound
	}
	return license, err
}

func (s *Store) Create(payload LicensePayload) (License, error) {
	data, err := applyPayload(License{
		Status:       "NOT_ACTIVATED",
		DurationDays: 30,
		MaxDevices:   1,
	}, payload, true)
	if err != nil {
		return License{}, err
	}

	license, err := scanLicense(s.db.QueryRow(
		`INSERT INTO licenses (key, product, owner, status, duration_days, max_devices)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+licenseColumns,
		data.Key, data.Product, data.Owner, data.Status, data.DurationDays, data.MaxDevices,
	))
	if err != nil {
		return License{}, mapDBError(err)
	}
	return license, nil
}

func (s *Store) Update(id int, payload LicensePayload) (License, error) {
	existing, err := s.Get(id)
	if err != nil {
		return License{}, err
	}

	data, err := applyPayload(existing, payload, false)
	if err != nil {
		return License{}, err
	}

	license, err := scanLicense(s.db.QueryRow(
		`UPDATE licenses
		 SET key = $1, product = $2, owner = $3, status = $4, duration_days = $5, max_devices = $6
		 WHERE id = $7
		 RETURNING `+licenseColumns,
		data.Key, data.Product, data.Owner, data.Status, data.DurationDays, data.MaxDevices, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return License{}, errNotFound
	}
	if err != nil {
		return License{}, mapDBError(err)
	}
	return license, nil
}

func (s *Store) Delete(id int) error {
	result, err := s.db.Exec("DELETE FROM licenses WHERE id = $1", id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound
	}
	return nil
}

func mapDBError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return errConflict
	}
	return err
}

func applyPayload(base License, payload LicensePayload, creating bool) (License, error) {
	out := base

	key, err := readString(payload.Key, "key", creating)
	if err != nil {
		return License{}, err
	}
	if payload.Key != nil {
		out.Key = key
	}

	product, err := readString(payload.Product, "product", creating)
	if err != nil {
		return License{}, err
	}
	if payload.Product != nil {
		out.Product = product
	}

	owner, err := readString(payload.Owner, "owner", creating)
	if err != nil {
		return License{}, err
	}
	if payload.Owner != nil {
		out.Owner = owner
	}

	if payload.Status != nil {
		if _, ok := allowedStatuses[*payload.Status]; !ok {
			return License{}, &validationError{msg: "status must be one of: NOT_ACTIVATED, ACTIVE, EXPIRED, BANNED"}
		}
		out.Status = *payload.Status
	}

	if payload.DurationDays != nil {
		if *payload.DurationDays < 1 {
			return License{}, &validationError{msg: "duration_days must be an integer >= 1"}
		}
		out.DurationDays = *payload.DurationDays
	}

	if payload.MaxDevices != nil {
		if *payload.MaxDevices < 1 {
			return License{}, &validationError{msg: "max_devices must be an integer >= 1"}
		}
		out.MaxDevices = *payload.MaxDevices
	}

	return out, nil
}

func readString(value *string, field string, required bool) (string, error) {
	if value == nil {
		if required {
			return "", &validationError{msg: field + " is required"}
		}
		return "", nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return "", &validationError{msg: field + " must not be empty"}
	}
	return trimmed, nil
}
