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

	"github.com/peterbourgon/ctxdata"
)

func TestFromEmptyChaining(t *testing.T) {
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
			method: "GetAll",
			exec:   func(d *ctxdata.Data) error { d.GetAll(); return nil },
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
			defer func() { json.NewEncoder(dst).Encode(d.GetAll()) }()
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
