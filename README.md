# ctxdata [![GoDoc](https://godoc.org/github.com/peterbourgon/ctxdata?status.svg)](https://godoc.org/github.com/peterbourgon/ctxdata/v2) [![builds.sr.ht status](https://builds.sr.ht/~peterbourgon/ctxdata.svg)](https://builds.sr.ht/~peterbourgon/ctxdata?)

A helper for collecting and emitting metadata throughout a request lifecycle.

When a new request arrives in your program, HTTP server, etc., use the New
constructor with the incoming request's context to construct a new, empty
Data.

```go
import "github.com/peterbourgon/ctxdata/v2"

func handler(w http.ResponseWriter, r *http.Request) {
    ctx, d := ctxdata.New(r.Context())
```

Use the returned context for all further operations on that request.

```go
    otherHandler(w, r.WithContext(ctx))
    result, err := process(ctx, r.Header.Get("X-My-Key"))
    // etc.
```

Whenever you want to add metadata to the request Data, 
use the From helper to fetch the Data from the context,
and call whatever method is appropriate.

```go
func otherHandler(w http.ResponseWriter, r *http.Request) {
    d := ctxdata.From(r.Context())
    d.Set("user", r.URL.Query().Get("user"))
    d.Set("corrleation_id", r.Header.Get("X-Correlation-ID"))
    // ...
}

func process(ctx context.Context, key string) {
    begin := time.Now()
    // ...
    ctxdata.From(ctx).Set("process_duration", time.Since(begin).String())
}
```

At the end of the request, all of the metadata collected throughout the
request's lifecycle will be available.

```go
    fmt.Fprintln(w, "OK")
    
    for k, v := range d {
        log.Printf("%s: %s", k, v)
    }
}
```

Here is an example middleware that writes a so-called wide event in JSON
to the dst at the end of each request.

```go
func logMiddleware(next http.Handler, dst io.Writer) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, d := ctxdata.New(r.Context())
        defer func() { json.NewEncoder(dst).Encode(d.Map()) }()
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```
