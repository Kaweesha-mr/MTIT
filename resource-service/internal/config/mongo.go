package config

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConnection struct {
	Client     *mongo.Client
	Collection *mongo.Collection
}

func ConnectMongo(cfg Config) (*MongoConnection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.MongoTimeoutSeconds)*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("failed to ping mongo: %w", err)
	}

	col := client.Database(cfg.DBName).Collection(cfg.CollectionName)

	return &MongoConnection{
		Client:     client,
		Collection: col,
	}, nil
}
