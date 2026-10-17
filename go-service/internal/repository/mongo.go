package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"node-go-labs/go-service/internal/model"
)

type MongoLicenseRepository struct {
	licenses *mongo.Collection
	counters *mongo.Collection
}

var _ LicenseRepository = (*MongoLicenseRepository)(nil)
var _ LicenseBatchReader = (*MongoLicenseRepository)(nil)

func NewMongoLicenseRepository(database *mongo.Database) (*MongoLicenseRepository, error) {
	repository := &MongoLicenseRepository{
		licenses: database.Collection("licenses"),
		counters: database.Collection("counters"),
	}
	ctx, cancel := dbContext()
	defer cancel()
	_, err := repository.licenses.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "key", Value: 1}}, Options: options.Index().SetUnique(true),
	})
	return repository, err
}

func (r *MongoLicenseRepository) List() ([]model.License, error) {
	return r.ListBatch(0, 1_000_000)
}

func (r *MongoLicenseRepository) FindByID(id int) (model.License, error) {
	ctx, cancel := dbContext()
	defer cancel()
	var license model.License
	err := r.licenses.FindOne(ctx, bson.M{"_id": id}).Decode(&license)
	return license, mapMongoError(err)
}

func (r *MongoLicenseRepository) FindByKey(key string) (model.License, error) {
	ctx, cancel := dbContext()
	defer cancel()
	var license model.License
	err := r.licenses.FindOne(ctx, bson.M{"key": key}).Decode(&license)
	return license, mapMongoError(err)
}

func (r *MongoLicenseRepository) nextID(ctx context.Context) (int, error) {
	var counter struct {
		Value int `bson:"value"`
	}
	err := r.counters.FindOneAndUpdate(ctx, bson.M{"_id": "licenses"}, bson.M{"$inc": bson.M{"value": 1}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&counter)
	return counter.Value, err
}

func (r *MongoLicenseRepository) Create(license model.License) (model.License, error) {
	ctx, cancel := dbContext()
	defer cancel()
	id, err := r.nextID(ctx)
	if err != nil {
		return model.License{}, err
	}
	license.ID = id
	_, err = r.licenses.InsertOne(ctx, license)
	return license, mapMongoError(err)
}

func (r *MongoLicenseRepository) Update(license model.License) (model.License, error) {
	ctx, cancel := dbContext()
	defer cancel()
	update := bson.M{"$set": bson.M{
		"key": license.Key, "product": license.Product, "owner": license.Owner,
		"status": license.Status, "duration_days": license.DurationDays, "max_devices": license.MaxDevices,
	}}
	var updated model.License
	err := r.licenses.FindOneAndUpdate(ctx, bson.M{"_id": license.ID}, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updated)
	return updated, mapMongoError(err)
}

func (r *MongoLicenseRepository) Delete(id int) error {
	ctx, cancel := dbContext()
	defer cancel()
	result, err := r.licenses.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MongoLicenseRepository) Count() (int, error) {
	ctx, cancel := dbContext()
	defer cancel()
	count, err := r.licenses.CountDocuments(ctx, bson.M{})
	return int(count), err
}

func (r *MongoLicenseRepository) ListBatch(afterID, limit int) ([]model.License, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cursor, err := r.licenses.Find(ctx, bson.M{"_id": bson.M{"$gt": afterID}},
		options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}).SetLimit(int64(limit)).SetBatchSize(int32(limit)))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var licenses []model.License
	if err := cursor.All(ctx, &licenses); err != nil {
		return nil, err
	}
	return licenses, nil
}

func mapMongoError(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	if mongo.IsDuplicateKeyError(err) {
		return ErrConflict
	}
	return err
}
