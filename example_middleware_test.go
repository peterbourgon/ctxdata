package ctxdata_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/peterbourgon/ctxdata/v3"
)

func ExampleNew_middleware() {
	// Let h be any normal HTTP handler.
	h := func(w http.ResponseWriter, r *http.Request) {
		// Extract a data from the context.
		d := ctxdata.From(r.Context())

		// Add information via Set.
		d.Set("method", r.Method)
		d.Set("path", r.URL.Path)
		d.Set("content_length", r.ContentLength)

		// Normal HTTP handler stuff.
		fmt.Fprint(w, "OK")
	}

	// In order for From to succeed in the handler, we first need to inject a
	// data via New. Typically, we do that with a middleware.
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Inject the data to the context.
			ctx, d := ctxdata.New(r.Context())

			// Use the returned context in the next handler.
			next.ServeHTTP(w, r.WithContext(ctx))

			// All metadata set in all downstream handlers is available to us.
			for _, kv := range d.GetAllSlice() {
				fmt.Printf("%s: %v\n", kv.Key, kv.Val)
			}
		})
	}

	// A mock server, request, and response recorder.
	var (
		server = middleware(http.HandlerFunc(h))
		req    = httptest.NewRequest("GET", "/path", strings.NewReader("request body"))
		rec    = httptest.NewRecorder()
	)
	server.ServeHTTP(rec, req)

	// Output:
	// method: GET
	// path: /path
	// content_length: 12
}
