package repository

import (
	"sort"
	"sync"

	"node-go-labs/lab4/go-service/internal/model"
)

type InMemoryLicenseRepository struct {
	mu       sync.RWMutex
	licenses map[int]model.License
	nextID   int
}

var _ LicenseRepository = (*InMemoryLicenseRepository)(nil)

func NewInMemoryLicenseRepository() *InMemoryLicenseRepository {
	return &InMemoryLicenseRepository{licenses: make(map[int]model.License), nextID: 1}
}

func (r *InMemoryLicenseRepository) List() ([]model.License, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	licenses := make([]model.License, 0, len(r.licenses))
	for _, license := range r.licenses {
		licenses = append(licenses, license)
	}
	sort.Slice(licenses, func(i, j int) bool { return licenses[i].ID < licenses[j].ID })
	return licenses, nil
}

func (r *InMemoryLicenseRepository) FindByID(id int) (model.License, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	license, ok := r.licenses[id]
	if !ok {
		return model.License{}, ErrNotFound
	}
	return license, nil
}

func (r *InMemoryLicenseRepository) FindByKey(key string) (model.License, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, license := range r.licenses {
		if license.Key == key {
			return license, nil
		}
	}
	return model.License{}, ErrNotFound
}

func (r *InMemoryLicenseRepository) Create(license model.License) (model.License, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	license.ID = r.nextID
	r.nextID++
	r.licenses[license.ID] = license
	return license, nil
}

func (r *InMemoryLicenseRepository) Update(license model.License) (model.License, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.licenses[license.ID]; !ok {
		return model.License{}, ErrNotFound
	}
	r.licenses[license.ID] = license
	return license, nil
}

func (r *InMemoryLicenseRepository) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.licenses[id]; !ok {
		return ErrNotFound
	}
	delete(r.licenses, id)
	return nil
}
