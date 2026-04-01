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
	Update(ctx context.Context, id int, item string, quantity int, unit string) error
	Delete(ctx context.Context, id int) error
	GetNextID(ctx context.Context) (int, error)
	List(ctx context.Context) ([]models.Resource, error)
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

func (r *MongoResourceRepository) Update(ctx context.Context, id int, item string, quantity int, unit string) error {
	res, err := r.collection.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": bson.M{"item": item, "quantity": quantity, "unit": unit, "available": quantity, "weight": fmt.Sprintf("%dkg", quantity), "updatedAt": time.Now().UTC()}},
	)
	if err != nil {
		return fmt.Errorf("update resource: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrResourceNotFound
	}

	return nil
}

func (r *MongoResourceRepository) Delete(ctx context.Context, id int) error {
	res, err := r.collection.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return fmt.Errorf("delete resource: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrResourceNotFound
	}

	return nil
}

func (r *MongoResourceRepository) List(ctx context.Context) ([]models.Resource, error) {
	cur, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	defer cur.Close(ctx)

	var resources []models.Resource
	for cur.Next(ctx) {
		var res models.Resource
		if err := cur.Decode(&res); err != nil {
			return nil, fmt.Errorf("decode resource: %w", err)
		}
		resources = append(resources, res)
	}

	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("list cursor: %w", err)
	}

	return resources, nil
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
