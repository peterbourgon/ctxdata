// Package ctxdata provides a helper type for request-scoped metadata.
package ctxdata

import (
	"context"
	"errors"
)

// KeyVal combines a string key with its abstract value into a single tuple.
// It's used internally, and as a return type for GetSlice.
type KeyVal struct {
	Key string
	Val interface{}
}

// Data is an opaque type that can be injected into a context at e.g. the start
// of a request, updated with metadata over the course of the request, and then
// queried at the end of the request.
//
// When a new request arrives in your program, HTTP server, etc., use the New
// constructor with the incoming request's context to construct a new, empty
// Data. Use the returned context for all further operations on that request.
// Use the From helper function to retrieve a previously-injected Data from a
// context, and set or get metadata. At the end of the request, all metadata
// collected will be available from any point in the callstack.
type Data struct {
	c chan []KeyVal
}

// New constructs a Data object and injects it into the provided context. Use
// the returned context for all further operations. The returned Data can be
// queried at any point for metadata collected over the life of the context.
func New(ctx context.Context) (context.Context, *Data) {
	c := make(chan []KeyVal, 1)
	c <- make([]KeyVal, 0, 32)
	d := &Data{c: c}
	return context.WithValue(ctx, contextKey{}, d), d
}

type contextKey struct{}

// From extracts a Data from the provided context. If no Data object was
// injected to the context, the returned pointer will be nil, but all Data
// methods gracefully handle this condition, so it's safe to consider the
// returned value always valid.
func From(ctx context.Context) *Data {
	v := ctx.Value(contextKey{})
	if v == nil {
		return nil
	}
	return v.(*Data)
}

// ErrNoData is returned by accessor methods when they're called on a nil
// pointer receiver. This typically means From was called on a context that
// didn't have a Data injected into it previously via New.
var ErrNoData = errors.New("no data in context")

// Set key to val. If key already exists, it will be overwritten. If this method
// is called on a nil Data pointer, it returns ErrNoData.
func (d *Data) Set(key string, val interface{}) error {
	if d == nil {
		return ErrNoData
	}

	s := <-d.c
	defer func() { d.c <- s }()

	for i := range s {
		if s[i].Key == key {
			s[i].Val = val
			s = append(s[:i], append(s[i+1:], s[i])...)
			return nil
		}
	}

	s = append(s, KeyVal{key, val})
	return nil
}

// ErrNotFound is returned by Get when the key isn't present.
var ErrNotFound = errors.New("key not found")

// Get the value associated with key, or return ErrNotFound. If this method is
// called on a nil Data pointer, it returns ErrNoData.
func (d *Data) Get(key string) (val interface{}, err error) {
	if d == nil {
		return nil, ErrNoData
	}

	s := <-d.c
	defer func() { d.c <- s }()

	for _, kv := range s {
		if kv.Key == key {
			return kv.Val, nil
		}
	}

	return nil, ErrNotFound
}

// GetAllSlice returns a slice of key/value pairs in the order in which they
// were set. If this method is called on a nil Data pointer, it returns
// ErrNoData.
func (d *Data) GetAllSlice() []KeyVal {
	if d == nil {
		return nil
	}

	s := <-d.c
	defer func() { d.c <- s }()

	r := make([]KeyVal, len(s))
	copy(r, s)
	return r
}

// GetAllMap returns a map of key to value. If this method is called on a nil
// Data pointer, it returns ErrNoData.
func (d *Data) GetAllMap() map[string]interface{} {
	if d == nil {
		return map[string]interface{}{}
	}

	s := <-d.c
	defer func() { d.c <- s }()

	m := make(map[string]interface{}, len(s))
	for _, kv := range s {
		m[kv.Key] = kv.Val
	}
	return m
}
