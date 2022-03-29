package consistent

// Consistent represents a hash sharding.
type Consistent interface {
	GetN(key string, n int) ([]string, error)
	Set(servers []string)
}
