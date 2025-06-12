package informer

// The Handler will handle the key.
type Handler interface {
	// Handle will handle the key.
	// Need for thead-safe.
	Handle(key string) error
}
