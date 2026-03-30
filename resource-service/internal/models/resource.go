package models

import "time"

const (
	ResourceStatusAvailable  = "AVAILABLE"
	ResourceStatusDispatched = "DISPATCHED"
)

type Resource struct {
	ID        int       `json:"id" bson:"id"`
	Item      string    `json:"item" bson:"item"`
	Quantity  int       `json:"quantity" bson:"quantity"`
	Unit      string    `json:"unit" bson:"unit"`
	Available int       `json:"available" bson:"available"`
	Weight    string    `json:"weight" bson:"weight"`
	Status    string    `json:"status" bson:"status"`
	CreatedAt time.Time `json:"createdAt,omitempty" bson:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" bson:"updatedAt,omitempty"`
}

type CreateResourceRequest struct {
	Item     string `json:"item"`
	Quantity int    `json:"quantity"`
	Unit     string `json:"unit"`
}

type DispatchRequest struct {
	ShelterID int `json:"shelterId"`
	Quantity  int `json:"quantity"`
}

type Shelter struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	CurrentOccupancy int    `json:"currentOccupancy"`
	MaxCapacity      int    `json:"maxCapacity"`
	Status           string `json:"status"`
}
