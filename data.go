// Package ctxdata provides a helper type for request-scoped metadata.
package ctxdata

import (
	"context"
	"errors"
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
	mtx sync.RWMutex
	dat map[string]string
}

// New constructs a Data object and injects it into the provided context. Use
// the returned context for all further operations in this request lifecycle.
// The returned Data object can be directly queried for metadata collected over
// the life of the request.
func New(ctx context.Context) (context.Context, *Data) {
	d := &Data{
		dat: map[string]string{},
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

	d.dat[key] = value
	return nil
}

// Get the value of key.
func (d *Data) Get(key string) (value string, ok bool) {
	if d == nil {
		return "", false
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()

	value, ok = d.dat[key]
	return value, ok
}

// GetDefault returns the value of key, or defaultValue if no value is
// available.
func (d *Data) GetDefault(key, defaultValue string) (value string) {
	if d == nil {
		return defaultValue
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()

	value, ok := d.dat[key]
	if !ok {
		return defaultValue
	}
	return value
}

// GetAll returns a copy of all of the keys and values currently stored.
func (d *Data) GetAll() (result map[string]string) {
	if d == nil {
		return nil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()

	result = make(map[string]string, len(d.dat))
	for k, v := range d.dat {
		result[k] = v
	}
	return result
}

// Walk calls fn for each key and value currently stored.
// If fn returns a non-nil error, Walk aborts with that error.
func (d *Data) Walk(fn func(key, value string) error) error {
	if d == nil {
		return ErrNilData
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()

	for k, v := range d.dat {
		if err := fn(k, v); err != nil {
			return err
		}
	}

	return nil
}
