package mongo

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Store redis store
type Store struct {
	Database   *mongo.Database
	Collection *mongo.Collection
	Option     Option
}

// Option redis option
type Option struct {
	Timeout time.Duration
}
