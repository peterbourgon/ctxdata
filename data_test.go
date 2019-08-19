package ctxdata_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
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
			ctxdata.From(r.Context()).Set("middleware_1", "b")
			next.ServeHTTP(w, r)
			ctxdata.From(r.Context()).Set("middleware_2", "c")
		})
	}(h)

	var results []string

	h = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, d := ctxdata.New(r.Context())
			d.Set("outer", "d")
			next.ServeHTTP(w, r.WithContext(ctx))
			d.Walk(func(key, value string) error {
				results = append(results, fmt.Sprintf("%s=%s", key, value))
				return nil
			})
		})
	}(h)

	r, w := httptest.NewRequest("GET", "/", nil), httptest.NewRecorder()
	h.ServeHTTP(w, r)
	sort.Strings(results)

	if want, have := []string{
		"inner=a",
		"middleware_1=b",
		"middleware_2=c",
		"outer=d",
	}, results; !reflect.DeepEqual(want, have) {
		t.Fatalf("want %v, have %v", want, have)
	}
}
