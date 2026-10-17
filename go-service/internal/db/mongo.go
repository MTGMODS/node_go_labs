package db

import (
	"context"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func OpenMongo() (*mongo.Client, *mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	maxPool := uint64(40)
	if parsed, err := strconv.ParseUint(os.Getenv("MONGO_POOL_MAX"), 10, 64); err == nil && parsed > 0 {
		maxPool = parsed
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")).SetMaxPoolSize(maxPool))
	if err != nil {
		return nil, nil, err
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, err
	}
	return client, client.Database(os.Getenv("MONGO_DB")), nil
}
