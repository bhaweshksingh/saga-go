package saga_test

import (
	"context"
	"fmt"
	"github.com/bhaweshksingh/saga-go"
)

// This example demonstrates how to use the Saga library.
func ExampleSaga() {
	s, abort := saga.New()

	s.AddStep(saga.Step{
		Execute: func(ctx context.Context) error {
			fmt.Println("Executing step 1")
			return nil
		},
		Compensate: func(ctx context.Context) error {
			fmt.Println("Compensating step 1")
			return nil
		},
	})

	s.AddStep(saga.Step{
		Execute: func(ctx context.Context) error {
			fmt.Println("Executing step 2")
			return fmt.Errorf("step 2 failed")
		},
		Compensate: func(ctx context.Context) error {
			fmt.Println("Compensating step 2")
			return nil
		},
	})

	var err error
	backgroundCtx := context.Background()
	defer func() {
		if err != nil {
			abort(backgroundCtx, &err)
		}
	}()

	err = s.Execute(backgroundCtx)
	if err != nil {
		if sagaErr, ok := err.(*saga.SagaError); ok {
			fmt.Printf("Saga failed: %v\n", sagaErr.OriginalError)
			if sagaErr.CompensationError != nil {
				fmt.Printf("Compensation errors: %v\n", sagaErr.CompensationError)
			}
		} else {
			fmt.Printf("Saga failed: %v\n", err)
		}
	}

	// Output:
	// Executing step 1
	// Executing step 2
	// Compensating step 1
	// Saga failed: step 2 failed
}
