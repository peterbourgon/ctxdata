// Package ctxdata provides a helper type for request-scoped metadata.
package ctxdata

import (
	"context"
	"errors"
	"sort"
	"sync"
)

// ErrNilData is returned by certain methods on the Data type when they're
// called with a nil pointer receiver. This can happen when methods are chained
// with the From helper, and indicates a Data object wasn't found in the
// context.
var ErrNilData = errors.New("nil data (no data in context?)")

type contextKey struct{}

// Data is an opaque type that can be injected into a context at the start of a
// request, updated with metadata over the course of the request, and queried at
// the end of the request for collected metadata.
//
// When a new request arrives in your program, HTTP server, etc., use the New
// constructor with the incoming request's context to construct a new, empty
// Data. Use the returned context for all further operations on that request,
// and use the From helper to retrieve the request's Data and set or get
// metadata. At the end of the request, all metadata collected will be available
// from any point in the callstack.
type Data struct {
	mtx   sync.RWMutex
	data  map[string]pair
	order int
}

type pair struct {
	s string
	i int
}

// New constructs a Data object and injects it into the provided context. Use
// the returned context for all further operations in this request lifecycle.
// The returned Data object can be directly queried for metadata collected over
// the life of the request.
func New(ctx context.Context) (context.Context, *Data) {
	d := &Data{
		data: map[string]pair{},
	}
	return context.WithValue(ctx, contextKey{}, d), d
}

// From extracts a Data object from the provided context. If no Data object was
// injected to the context, the returned pointer will be nil, but all Data
// methods gracefully handle this situation, so it's safe to chain.
func From(ctx context.Context) *Data {
	v := ctx.Value(contextKey{})
	if v == nil {
		return nil
	}
	return v.(*Data)
}

// Set key to value.
func (d *Data) Set(key, value string) error {
	if d == nil {
		return ErrNilData
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()

	d.data[key] = pair{value, d.order}
	d.order++

	return nil
}

// Get the value of key.
func (d *Data) Get(key string) (value string, ok bool) {
	if d == nil {
		return "", false
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()

	p, ok := d.data[key]
	return p.s, ok
}

// GetDefault returns the value of key, or defaultValue if no value is
// available.
func (d *Data) GetDefault(key, defaultValue string) (value string) {
	if d == nil {
		return defaultValue
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()

	p, ok := d.data[key]
	if !ok {
		return defaultValue
	}
	return p.s
}

// GetAll returns a copy of all of the keys and values currently stored
// as an unordered map.
func (d *Data) GetAll() map[string]string {
	if d == nil {
		return nil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()

	result := make(map[string]string, len(d.data))
	for k, p := range d.data {
		result[k] = p.s
	}

	return result
}

// KeyValue simply combines a key and its value.
type KeyValue struct {
	Key   string
	Value string
}

// GetAllSlice returns a copy of all of the keys and values currently stored as
// a slice of KeyValues, in the order they were added (set).
func (d *Data) GetAllSlice() []KeyValue {
	if d == nil {
		return nil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()

	result := make([]KeyValue, len(d.data))
	for i, k := range d.orderedKeys() {
		result[i] = KeyValue{k, d.data[k].s}
	}

	return result
}

// Walk calls fn for each key and value currently stored in the order they were
// added (set). If fn returns a non-nil error, Walk aborts with that error.
func (d *Data) Walk(fn func(key, value string) error) error {
	if d == nil {
		return ErrNilData
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()

	for _, k := range d.orderedKeys() {
		if err := fn(k, d.data[k].s); err != nil {
			return err
		}
	}

	return nil
}

func (d *Data) orderedKeys() []string {
	intermediate := make([]pair, 0, len(d.data))
	for k, p := range d.data {
		intermediate = append(intermediate, pair{k, p.i})
	}

	sort.Slice(intermediate, func(i, j int) bool {
		return intermediate[i].i < intermediate[j].i
	})

	result := make([]string, len(intermediate))
	for i, p := range intermediate {
		result[i] = p.s
	}

	return result
}
