package ctxdata_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/peterbourgon/ctxdata/v2"
)

func TestFromEmptyChaining(t *testing.T) {
	t.Parallel()

	for _, testcase := range []struct {
		method string
		exec   func(*ctxdata.Data) error
		want   error
	}{
		{
			method: "Set",
			exec:   func(d *ctxdata.Data) error { return d.Set("k", "v") },
			want:   ctxdata.ErrNilData,
		},
		{
			method: "Get",
			exec:   func(d *ctxdata.Data) error { d.Get("k"); return nil },
			want:   nil,
		},
		{
			method: "GetDefault",
			exec:   func(d *ctxdata.Data) error { d.GetDefault("k", "def"); return nil },
			want:   nil,
		},
		{
			method: "Map",
			exec:   func(d *ctxdata.Data) error { d.Map(); return nil },
			want:   nil,
		},
		{
			method: "Slice",
			exec:   func(d *ctxdata.Data) error { d.Slice(); return nil },
			want:   nil,
		},
		{
			method: "Walk",
			exec:   func(d *ctxdata.Data) error { return d.Walk(func(string, string) error { return nil }) },
			want:   ctxdata.ErrNilData,
		},
	} {
		t.Run(testcase.method, func(t *testing.T) {
			if want, have := testcase.want, testcase.exec(ctxdata.From(context.Background())); want != have {
				t.Fatalf("want %v, have %v", want, have)
			}
		})
	}
}

func TestCallstack(t *testing.T) {
	t.Parallel()

	var h http.Handler

	h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxdata.From(r.Context()).Set("inner", "a")
		fmt.Fprintln(w, "OK")
	})

	h = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctxdata.From(r.Context()).Set("middleware", "b")
			next.ServeHTTP(w, r)
		})
	}(h)

	var buf bytes.Buffer

	h = func(next http.Handler, dst io.Writer) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, d := ctxdata.New(r.Context())
			defer func() { json.NewEncoder(dst).Encode(d.Map()) }()
			next.ServeHTTP(w, r.WithContext(ctx))
			d.Set("outer", "c")
		})
	}(h, &buf)

	r, w := httptest.NewRequest("GET", "/", nil), httptest.NewRecorder()
	h.ServeHTTP(w, r)

	want := map[string]string{"inner": "a", "middleware": "b", "outer": "c"}
	var have map[string]string
	json.NewDecoder(&buf).Decode(&have)
	if !reflect.DeepEqual(want, have) {
		t.Fatalf("want %v, have %v", want, have)
	}
}

func TestOrder(t *testing.T) {
	t.Parallel()

	_, d := ctxdata.New(context.Background())

	d.Set("a", "1")
	d.Set("b", "2")
	d.Set("c", "3")
	d.Set("a", "4")
	d.Set("d", "5")

	want := []ctxdata.KeyValue{
		{"b", "2"},
		{"c", "3"},
		{"a", "4"},
		{"d", "5"},
	}

	if have := d.Slice(); !reflect.DeepEqual(want, have) {
		t.Fatalf("want %v, have %v", want, have)
	}
}
