# ctxdata [![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/peterbourgon/ctxdata/v4) [![Build Status](https://img.shields.io/endpoint.svg?url=https%3A%2F%2Factions-badge.atrox.dev%2Fpeterbourgon%2Fctxdata%2Fbadge&style=flat-square&label=build)](https://github.com/peterbourgon/ctxdata/actions?query=workflow%3ATest)

A helper for collecting and emitting metadata throughout a request lifecycle.

When a new request arrives in your program, HTTP server, etc., create a new Data
value and inject it into the request context via
[ctxdata.New](https://pkg.go.dev/github.com/peterbourgon/ctxdata/v4#New).

```go
import "github.com/peterbourgon/ctxdata/v4"

func handler(w http.ResponseWriter, r *http.Request) {
    ctx, d := ctxdata.New(r.Context())
```

Use the returned context for all further operations on that request.

```go
    otherHandler(w, r.WithContext(ctx))
    result, err := process(ctx, r.Header.Get("X-My-Key"))
    // etc.
```

Once the Data has been created and injected to the context, use
[ctxdata.From](https://pkg.go.dev/github.com/peterbourgon/ctxdata/v4#From)
from any downstream function with access to the context to fetch it and add more
metadata about the request.

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
    ctxdata.From(ctx).Set("process_duration", time.Since(begin))
}
```

At the end of the request, all of the metadata collected throughout the
request's lifecycle will be available.

```go
    fmt.Fprintln(w, "OK")

    for _, kv := range d.GetAllSlice() {
        log.Printf("%s: %s", kv.Key, kv.Val)
    }
}
```

Here is an example middleware that writes a so-called wide event in JSON
to the dst at the end of each request.

```go
func logMiddleware(next http.Handler, dst io.Writer) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, d := ctxdata.New(r.Context())
        defer func() { json.NewEncoder(dst).Encode(d.GetAllMap()) }()
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```
