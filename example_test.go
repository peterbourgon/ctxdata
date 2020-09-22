package ctxdata_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/peterbourgon/ctxdata/v4"
)

// This example demonstrates how to use GetAs to retrieve metadata into an
// arbitrary type.
func ExampleData_GetAs() {
	type DomainError struct {
		Code   int
		Reason string
	}

	_, d := ctxdata.New(context.Background())
	derr := DomainError{Code: http.StatusTeapot, Reason: "Earl Gray exception"}
	d.Set("err", derr)

	if target := (DomainError{}); d.GetAs("err", &target) == nil {
		fmt.Printf("DomainError Code=%d Reason=%q\n", target.Code, target.Reason)
	}

	// Output:
	// DomainError Code=418 Reason="Earl Gray exception"
}
