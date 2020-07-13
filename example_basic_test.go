package ctxdata_test

import (
	"context"
	"fmt"
	"time"

	"github.com/peterbourgon/ctxdata/v3"
)

func ExampleNew_basic() {
	// Let f be any function that takes a context.
	f := func(ctx context.Context) {
		// Use ctxdata.From on the context to extract a ctxdata.Data.
		d := ctxdata.From(ctx)

		// Add information via Set.
		d.Set("foo", "hello")
		d.Set("bar", 12345)
		d.Set("baz", 34*time.Second)
	}

	// In order for those Set calls to be effective,
	// inject a ctxdata.Data into the context.
	ctx, d := ctxdata.New(context.Background())

	// Now we can call function f with the returned context.
	f(ctx)

	// Afterwards, everything that was Set is available to us.
	fmt.Printf("foo: %v\n", d.GetString("foo", "default value"))
	fmt.Printf("bar: %v\n", d.GetInt("bar", 0))
	fmt.Printf("baz: %v\n", d.GetDuration("baz", time.Hour))

	// Output:
	// foo: hello
	// bar: 12345
	// baz: 34s
}
