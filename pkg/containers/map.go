package containers

import (
	"errors"
	"fmt"
)

// ErrNotFound is wrapped and returned when an item is
// not found in a MapStore instance.
var ErrNotFound = errors.New("not found")

// MapStore is a map with an accessor method Get which
// returns an error when the item is not present in the
// underlying map, with a core error of ErrNotFound.
type MapStore[K comparable, V any] map[K]V

// Get accesses the map with the provided key k and returns
// a ErrNotFound wrapped error if it is not present.
func (m MapStore[K, V]) Get(k K) (v V, err error) {
	v, ok := m[k]
	if !ok {
		err = fmt.Errorf(`key "%v": %w`, k, ErrNotFound)
		return
	}

	return v, nil
}
