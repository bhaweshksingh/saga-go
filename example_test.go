package saga_test

import (
	"context"
	"fmt"
	"github.com/bhaweshksingh/saga-go"
)

// This example demonstrates how to use the Saga library.
func ExampleSaga() {
	s := saga.New()

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

	err := s.Execute(context.Background())
	if err != nil {
		fmt.Printf("Saga failed: %v\n", err)
	}

	// Output:
	// Executing step 1
	// Executing step 2
	// Compensating step 1
	// Saga failed: saga failed: step 2 failed
}
