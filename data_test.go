package ctxdata_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/peterbourgon/ctxdata/v3"
)

func TestBasics(t *testing.T) {
	t.Parallel()

	ctx, d := ctxdata.New(context.Background())
	d.Set("a", 1)

	{
		d := ctxdata.From(ctx)
		d.Set("b", 2)
	}

	{
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		d := ctxdata.From(ctx)
		d.Set("a", 3)
		d.Set("c", 4)
	}

	{
		s := d.GetAllSlice()

		if want, have := 3, len(s); want != have {
			t.Fatalf("len: want %d, have %d", want, have)
		}

		for i, want := range []struct {
			Key string
			Val int
		}{
			{"b", 2},
			{"a", 3},
			{"c", 4},
		} {
			if want, have := want.Key, s[i].Key; want != have {
				t.Errorf("s[%d]: Key: want %q, have %q", i, want, have)
			}
			if want, have := want.Val, s[i].Val.(int); want != have {
				t.Errorf("s[%d]: Val: want %q, have %q", i, want, have)
			}
		}
	}

	{
		_, err := d.Get("foo")
		if want, have := ctxdata.ErrNotFound, err; !errors.Is(have, want) {
			t.Errorf("Get(foo): want error %v, have %v", want, have)
		}

		v, err := d.Get("a")
		if err != nil {
			t.Fatalf("Get(a): unexpected error %v", err)
		}
		if want, have := 3, v.(int); want != have {
			t.Errorf("Get(a): Val: want %d, have %d", want, have)
		}
	}
}

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
			want:   ctxdata.ErrNoData,
		},
		{
			method: "Get",
			exec:   func(d *ctxdata.Data) error { d.Get("k"); return nil },
			want:   nil,
		},
		{
			method: "GetAllSlice",
			exec:   func(d *ctxdata.Data) error { d.GetAllSlice(); return nil },
			want:   nil,
		},
		{
			method: "GetAllMap",
			exec:   func(d *ctxdata.Data) error { d.GetAllMap(); return nil },
			want:   nil,
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
			defer func() { json.NewEncoder(dst).Encode(d.GetAllMap()) }()
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

	want := []ctxdata.KeyVal{
		{"b", "2"},
		{"c", "3"},
		{"a", "4"},
		{"d", "5"},
	}

	have := d.GetAllSlice()

	if want, have := len(want), len(have); want != have {
		t.Fatalf("len: want %d, have %d", want, have)
	}

	for i := range have {
		if want, have := want[i].Key, have[i].Key; want != have {
			t.Errorf("%d: Key: want %q, have %q", i+1, want, have)
			continue
		}

		s, ok := have[i].Val.(string)
		if !ok {
			t.Errorf("%d: Val: not a string", i+1)
			continue
		}

		if want, have := want[i].Val, s; want != have {
			t.Errorf("%d: Val: want %q, have %q", i+1, want, have)
		}
	}
}

func TestConcurrency(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	ctx, d := ctxdata.New(ctx)

	worker := func(ctx context.Context) {
		d := ctxdata.From(ctx)
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		var counter uint64
		for {
			select {
			case <-ticker.C:
				d.Set("n", counter)
				counter++
			case <-ctx.Done():
				return
			}
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctx)
		}()
	}

	time.Sleep(3 * time.Second)
	cancel()
	wg.Wait()

	t.Logf("n: %d", d.GetUint64("n", 0))
}

func TestGetAs(t *testing.T) {
	t.Parallel()

	type foo struct {
		s string
		i int
	}

	_, d := ctxdata.New(context.Background())
	d.Set("foo", foo{s: "hello", i: 101})

	var f foo
	if want, have := error(nil), d.GetAs("foo", &f); want != have {
		t.Fatalf("GetAs(foo, &f): want error %v, have %v", want, have)
	}
	if want, have := "hello", f.s; want != have {
		t.Errorf("f.s: want %q, have %q", want, have)
	}
	if want, have := 101, f.i; want != have {
		t.Errorf("f.i: want %d, have %d", want, have)
	}

	type bar struct {
		_ string
		_ int
	}
	var b bar
	if want, have := ctxdata.ErrIncompatibleType, d.GetAs("foo", &b); want != have {
		t.Fatalf("GetAs(foo, &b): want %v, have %v", want, have)
	}

	type baz struct {
		_ string
		_ int
		_ float64
	}
	var z baz
	if want, have := ctxdata.ErrIncompatibleType, d.GetAs("foo", &z); want != have {
		t.Fatalf("GetAs(foo, &z): want %v, have %v", want, have)
	}
}
