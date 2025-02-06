package informer

// Reflector will reflect the keys.
type Reflector interface {
	// Watch will watch the keys.
	// No need for thead-safe.
	Watch() ([]string, error)
	// Get will get the value of the key.
	// Need for thead-safe.
	Get(key string) (interface{}, bool)
}
