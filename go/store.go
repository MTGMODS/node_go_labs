package main

import (
	"errors"
	"sort"
	"strings"
	"sync"
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
	// Як dict, але з lock: у Go кожен HTTP-запит іде в окремій goroutine.
	mu       sync.RWMutex
	licenses map[int]License
	nextID   int
}

func NewStore() *Store {
	s := &Store{
		licenses: make(map[int]License),
		nextID:   2,
	}
	s.licenses[1] = License{
		ID:           1,
		Key:          "MTGM-VIP1-AAAA-0001",
		Product:      "MTG MODS VIP",
		Owner:        "bogdan",
		Status:       "ACTIVE",
		DurationDays: 30,
		MaxDevices:   2,
	}
	return s
}

func (s *Store) List() []License {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]License, 0, len(s.licenses))
	for _, license := range s.licenses {
		items = append(items, license)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

func (s *Store) Get(id int) (License, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	license, ok := s.licenses[id]
	if !ok {
		return License{}, errNotFound
	}
	return license, nil
}

func (s *Store) Create(payload LicensePayload) (License, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := applyPayload(License{
		Status:       "NOT_ACTIVATED",
		DurationDays: 30,
		MaxDevices:   1,
	}, payload, true)
	if err != nil {
		return License{}, err
	}
	if err := s.assertUniqueKey(data.Key, 0); err != nil {
		return License{}, err
	}

	data.ID = s.nextID
	s.nextID++
	s.licenses[data.ID] = data
	return data, nil
}

func (s *Store) Update(id int, payload LicensePayload) (License, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.licenses[id]
	if !ok {
		return License{}, errNotFound
	}

	data, err := applyPayload(existing, payload, false)
	if err != nil {
		return License{}, err
	}
	if err := s.assertUniqueKey(data.Key, id); err != nil {
		return License{}, err
	}

	s.licenses[id] = data
	return data, nil
}

func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.licenses[id]; !ok {
		return errNotFound
	}
	delete(s.licenses, id)
	return nil
}

func (s *Store) assertUniqueKey(key string, currentID int) error {
	for _, license := range s.licenses {
		if license.Key == key && license.ID != currentID {
			return errConflict
		}
	}
	return nil
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
