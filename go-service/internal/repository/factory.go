package repository

import (
	"database/sql"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

func NewFromType(repositoryType string, postgres *sql.DB, mongoDB *mongo.Database) (LicenseRepository, error) {
	switch repositoryType {
	case "postgres":
		return NewPostgresLicenseRepository(postgres), nil
	case "mongo":
		return NewMongoLicenseRepository(mongoDB)
	default:
		return nil, fmt.Errorf("unsupported REPO_TYPE %q: expected postgres or mongo", repositoryType)
	}
}
