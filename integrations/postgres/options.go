package postgres

import "github.com/google/uuid"

// IDGenerator is a function that generates unique IDs for database records.
type IDGenerator func() string

type storeOptions struct {
	idGenerator IDGenerator
}

// Option configures a postgres store.
type Option func(*storeOptions)

// WithIDGenerator sets a custom ID generator for the store.
// By default, UUIDs are used.
func WithIDGenerator(gen IDGenerator) Option {
	return func(o *storeOptions) {
		o.idGenerator = gen
	}
}

func defaultOptions() storeOptions {
	return storeOptions{
		idGenerator: func() string {
			return uuid.New().String()
		},
	}
}
