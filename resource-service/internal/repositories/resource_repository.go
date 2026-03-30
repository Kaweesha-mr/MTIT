package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"resource-service/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrResourceNotFound = errors.New("resource not found")

type ResourceRepository interface {
	Create(ctx context.Context, resource *models.Resource) error
	GetByID(ctx context.Context, id int) (*models.Resource, error)
	UpdateDispatch(ctx context.Context, id int, available int, status string) error
	GetNextID(ctx context.Context) (int, error)
}

type MongoResourceRepository struct {
	collection *mongo.Collection
	counters   *mongo.Collection
}

func NewMongoResourceRepository(db *mongo.Database, collectionName string) *MongoResourceRepository {
	return &MongoResourceRepository{
		collection: db.Collection(collectionName),
		counters:   db.Collection("counters"),
	}
}

func (r *MongoResourceRepository) Create(ctx context.Context, resource *models.Resource) error {
	now := time.Now().UTC()
	resource.CreatedAt = now
	resource.UpdatedAt = now

	_, err := r.collection.InsertOne(ctx, resource)
	if err != nil {
		return fmt.Errorf("insert resource: %w", err)
	}

	return nil
}

func (r *MongoResourceRepository) GetByID(ctx context.Context, id int) (*models.Resource, error) {
	var resource models.Resource
	err := r.collection.FindOne(ctx, bson.M{"id": id}).Decode(&resource)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrResourceNotFound
		}
		return nil, fmt.Errorf("find resource by id: %w", err)
	}

	return &resource, nil
}

func (r *MongoResourceRepository) UpdateDispatch(ctx context.Context, id int, available int, status string) error {
	res, err := r.collection.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": bson.M{"available": available, "status": status, "updatedAt": time.Now().UTC()}},
	)
	if err != nil {
		return fmt.Errorf("update dispatch: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrResourceNotFound
	}

	return nil
}

func (r *MongoResourceRepository) GetNextID(ctx context.Context) (int, error) {
	after := options.After
	res := r.counters.FindOneAndUpdate(
		ctx,
		bson.M{"_id": "resourceId"},
		bson.M{"$inc": bson.M{"sequence": 1}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(after),
	)

	if res.Err() != nil {
		return 0, fmt.Errorf("increment resource id counter: %w", res.Err())
	}

	var counter struct {
		Sequence int `bson:"sequence"`
	}

	if err := res.Decode(&counter); err != nil {
		return 0, fmt.Errorf("decode counter result: %w", err)
	}

	return counter.Sequence, nil
}
