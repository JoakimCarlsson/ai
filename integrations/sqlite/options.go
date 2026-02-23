package sqlite

type storeOptions struct {
	tablePrefix string
}

// Option configures a sqlite store.
type Option func(*storeOptions)

// WithTablePrefix sets a prefix for all table names created by the store.
// For example, WithTablePrefix("chat_") creates "chat_sessions" and "chat_messages"
// instead of "sessions" and "messages".
func WithTablePrefix(prefix string) Option {
	return func(o *storeOptions) {
		o.tablePrefix = prefix
	}
}

func defaultOptions() storeOptions {
	return storeOptions{}
}
