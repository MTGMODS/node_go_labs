package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"node-go-labs/go-service/internal/model"
)

type PostgresLicenseRepository struct {
	db *sql.DB
}

var _ LicenseRepository = (*PostgresLicenseRepository)(nil)
var _ LicenseBatchReader = (*PostgresLicenseRepository)(nil)

func NewPostgresLicenseRepository(database *sql.DB) *PostgresLicenseRepository {
	return &PostgresLicenseRepository{db: database}
}

const licenseColumns = "id, license_key, product, owner, status, duration_days, max_devices"

type scanner interface {
	Scan(dest ...any) error
}

func scanLicense(row scanner) (model.License, error) {
	var license model.License
	err := row.Scan(&license.ID, &license.Key, &license.Product, &license.Owner,
		&license.Status, &license.DurationDays, &license.MaxDevices)
	return license, err
}

func dbContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (r *PostgresLicenseRepository) List() ([]model.License, error) {
	return r.ListBatch(0, 1_000_000)
}

func (r *PostgresLicenseRepository) FindByID(id int) (model.License, error) {
	ctx, cancel := dbContext()
	defer cancel()
	license, err := scanLicense(r.db.QueryRowContext(ctx, "SELECT "+licenseColumns+" FROM licenses WHERE id=$1", id))
	if errors.Is(err, sql.ErrNoRows) {
		return model.License{}, ErrNotFound
	}
	return license, err
}

func (r *PostgresLicenseRepository) FindByKey(key string) (model.License, error) {
	ctx, cancel := dbContext()
	defer cancel()
	license, err := scanLicense(r.db.QueryRowContext(ctx, "SELECT "+licenseColumns+" FROM licenses WHERE license_key=$1", key))
	if errors.Is(err, sql.ErrNoRows) {
		return model.License{}, ErrNotFound
	}
	return license, err
}

func (r *PostgresLicenseRepository) Create(license model.License) (model.License, error) {
	ctx, cancel := dbContext()
	defer cancel()
	created, err := scanLicense(r.db.QueryRowContext(ctx,
		`INSERT INTO licenses (license_key, product, owner, status, duration_days, max_devices)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING `+licenseColumns,
		license.Key, license.Product, license.Owner, license.Status, license.DurationDays, license.MaxDevices))
	return created, mapPostgresError(err)
}

func (r *PostgresLicenseRepository) Update(license model.License) (model.License, error) {
	ctx, cancel := dbContext()
	defer cancel()
	updated, err := scanLicense(r.db.QueryRowContext(ctx,
		`UPDATE licenses SET license_key=$2, product=$3, owner=$4, status=$5,
		 duration_days=$6, max_devices=$7, updated_at=NOW() WHERE id=$1 RETURNING `+licenseColumns,
		license.ID, license.Key, license.Product, license.Owner, license.Status, license.DurationDays, license.MaxDevices))
	if errors.Is(err, sql.ErrNoRows) {
		return model.License{}, ErrNotFound
	}
	return updated, mapPostgresError(err)
}

func (r *PostgresLicenseRepository) Delete(id int) error {
	ctx, cancel := dbContext()
	defer cancel()
	result, err := r.db.ExecContext(ctx, "DELETE FROM licenses WHERE id=$1", id)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresLicenseRepository) Count() (int, error) {
	ctx, cancel := dbContext()
	defer cancel()
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM licenses").Scan(&count)
	return count, err
}

func (r *PostgresLicenseRepository) ListBatch(afterID, limit int) ([]model.License, error) {
	ctx, cancel := dbContext()
	defer cancel()
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+licenseColumns+" FROM licenses WHERE id>$1 ORDER BY id LIMIT $2", afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	licenses := make([]model.License, 0, limit)
	for rows.Next() {
		license, scanErr := scanLicense(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		licenses = append(licenses, license)
	}
	return licenses, rows.Err()
}

func mapPostgresError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return ErrConflict
	}
	return err
}
