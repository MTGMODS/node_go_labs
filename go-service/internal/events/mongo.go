package events

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"node-go-labs/go-service/internal/model"
	"node-go-labs/go-service/internal/service"
)

type StoredEvent struct {
	ID        string        `json:"id"`
	Type      string        `json:"type"`
	Data      model.License `json:"data"`
	Timestamp time.Time     `json:"timestamp"`
	Runtime   string        `json:"runtime"`
}

type MongoLogger struct {
	collection *mongo.Collection
}

func NewMongoLogger(database *mongo.Database) *MongoLogger {
	return &MongoLogger{collection: database.Collection("events")}
}

func (l *MongoLogger) Log(event service.Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := l.collection.InsertOne(ctx, event)
	return err
}

func (l *MongoLogger) List(limit int) ([]StoredEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cursor, err := l.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var documents []struct {
		ID        primitive.ObjectID `bson:"_id"`
		Type      string             `bson:"type"`
		Data      model.License      `bson:"data"`
		Timestamp time.Time          `bson:"timestamp"`
		Runtime   string             `bson:"runtime"`
	}
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}
	result := make([]StoredEvent, 0, len(documents))
	for _, document := range documents {
		result = append(result, StoredEvent{
			ID: document.ID.Hex(), Type: document.Type, Data: document.Data,
			Timestamp: document.Timestamp, Runtime: document.Runtime,
		})
	}
	return result, nil
}
