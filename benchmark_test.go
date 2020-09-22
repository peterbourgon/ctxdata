package ctxdata_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/peterbourgon/ctxdata/v4"
)

func BenchmarkMiddleware(b *testing.B) {
	b.Run("no ctxdata", func(b *testing.B) {
		h := http.HandlerFunc(nopHandler)
		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/path", nil)
			h.ServeHTTP(rec, req)
		}
	})

	b.Run("with ctxdata", func(b *testing.B) {
		h := middleware(http.HandlerFunc(dataHandler))
		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/path", nil)
			h.ServeHTTP(rec, req)
		}
	})
}

func nopHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	d := ctxdata.From(r.Context())
	d.Set("method", r.Method)
	d.Set("path", r.URL.Path)
	d.Set("duration", 123*time.Microsecond)
	fmt.Fprintln(w, "OK")
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, d := ctxdata.New(r.Context())

		defer func() {
			for _, kv := range d.GetAllSlice() {
				fmt.Fprintf(ioutil.Discard, "%s: %v\n", kv.Key, kv.Key)
			}
		}()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
