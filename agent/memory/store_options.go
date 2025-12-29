package memory

import "github.com/google/uuid"

// IDGenerator is a function that generates unique IDs for memory entries.
type IDGenerator func() string

// storedEntry holds a memory entry along with its vector embedding.
type storedEntry struct {
	Entry
	Vector []float32 `json:"vector"`
}

type storeConfig struct {
	idGenerator IDGenerator
}

// StoreOption configures a built-in memory store.
type StoreOption func(*storeConfig)

// WithIDGenerator sets a custom ID generator for the store.
// By default, UUIDs are used.
func WithIDGenerator(gen IDGenerator) StoreOption {
	return func(c *storeConfig) {
		c.idGenerator = gen
	}
}

func defaultStoreConfig() storeConfig {
	return storeConfig{
		idGenerator: func() string {
			return uuid.New().String()
		},
	}
}
