package mongo

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Store mongo store
type Store struct {
	Database   *mongo.Database
	Collection *mongo.Collection
	Option     Option
}

// Option mongo store option
type Option struct {
	Timeout time.Duration
	Prefix  string // Key prefix for namespacing
}
