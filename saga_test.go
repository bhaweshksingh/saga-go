package saga_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bhaweshksingh/saga-go"
	"github.com/stretchr/testify/assert"
)

func TestSaga(t *testing.T) {
	t.Run("Successful execution", func(t *testing.T) {
		s, _ := saga.New()
		steps := []string{}

		s.AddStep(saga.Step{
			Execute: func(ctx context.Context) error {
				steps = append(steps, "step1")
				return nil
			},
			Compensate: func(ctx context.Context) error {
				steps = append(steps, "compensate1")
				return nil
			},
		})

		s.AddStep(saga.Step{
			Execute: func(ctx context.Context) error {
				steps = append(steps, "step2")
				return nil
			},
			Compensate: func(ctx context.Context) error {
				steps = append(steps, "compensate2")
				return nil
			},
		})

		err := s.Execute(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, []string{"step1", "step2"}, steps)
	})

	t.Run("Failure in second step", func(t *testing.T) {
		s, _ := saga.New()
		steps := []string{}

		s.AddStep(saga.Step{
			Execute: func(ctx context.Context) error {
				steps = append(steps, "step1")
				return nil
			},
			Compensate: func(ctx context.Context) error {
				steps = append(steps, "compensate1")
				return nil
			},
		})

		s.AddStep(saga.Step{
			Execute: func(ctx context.Context) error {
				steps = append(steps, "step2")
				return errors.New("step 2 failed")
			},
			Compensate: func(ctx context.Context) error {
				steps = append(steps, "compensate2")
				return nil
			},
		})

		err := s.Execute(context.Background())
		assert.Error(t, err)
		assert.Equal(t, []string{"step1", "step2", "compensate1"}, steps)
		assert.Contains(t, err.Error(), "step 2 failed")
	})

	t.Run("Compensation error", func(t *testing.T) {
		s, _ := saga.New()
		steps := []string{}

		s.AddStep(saga.Step{
			Execute: func(ctx context.Context) error {
				steps = append(steps, "step1")
				return nil
			},
			Compensate: func(ctx context.Context) error {
				steps = append(steps, "compensate1")
				return errors.New("compensation 1 failed")
			},
		})

		s.AddStep(saga.Step{
			Execute: func(ctx context.Context) error {
				steps = append(steps, "step2")
				return errors.New("step 2 failed")
			},
			Compensate: func(ctx context.Context) error {
				steps = append(steps, "compensate2")
				return nil
			},
		})

		err := s.Execute(context.Background())
		assert.Error(t, err)
		sagaErr, ok := err.(*saga.SagaError)
		assert.True(t, ok, "Expected SagaError")
		assert.Contains(t, sagaErr.OriginalError.Error(), "step 2 failed")
		assert.Contains(t, sagaErr.CompensationError.Error(), "compensation 1 failed")
		assert.Equal(t, []string{"step1", "step2", "compensate1"}, steps)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		s, _ := saga.New()
		steps := []string{}
		var mu sync.Mutex

		addStep := func(name string) {
			mu.Lock()
			defer mu.Unlock()
			steps = append(steps, name)
		}

		s.AddStep(saga.Step{
			Execute: func(ctx context.Context) error {
				addStep("step1")
				return nil
			},
			Compensate: func(ctx context.Context) error {
				addStep("compensate1")
				return nil
			},
		})

		s.AddStep(saga.Step{
			Execute: func(ctx context.Context) error {
				time.Sleep(100 * time.Millisecond)
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					addStep("step2")
					return nil
				}
			},
			Compensate: func(ctx context.Context) error {
				addStep("compensate2")
				return nil
			},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := s.Execute(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")

		mu.Lock()
		defer mu.Unlock()
		assert.Contains(t, steps, "step1")
		assert.Contains(t, steps, "compensate1")
		assert.NotContains(t, steps, "step2")
		assert.NotContains(t, steps, "compensate2")
	})

	t.Run("Manual abort", func(t *testing.T) {
		s, abortFunc := saga.New()
		steps := []string{}

		s.AddStep(saga.Step{
			Execute: func(ctx context.Context) error {
				steps = append(steps, "step1")
				return nil
			},
			Compensate: func(ctx context.Context) error {
				steps = append(steps, "compensate1")
				return nil
			},
		})

		s.AddStep(saga.Step{
			Execute: func(ctx context.Context) error {
				steps = append(steps, "step2")
				return nil
			},
			Compensate: func(ctx context.Context) error {
				steps = append(steps, "compensate2")
				return nil
			},
		})

		backgroundCtx := context.Background()
		go func() {
			time.Sleep(50 * time.Millisecond)
			var err error
			abortFunc(backgroundCtx, &err)
		}()

		err := s.Execute(backgroundCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction rollback")
		assert.Contains(t, steps, "compensate1")
		assert.Contains(t, steps, "compensate2")
	})

	t.Run("Adding step after execution", func(t *testing.T) {
		s, _ := saga.New()
		steps := []string{}

		s.AddStep(saga.Step{
			Execute: func(ctx context.Context) error {
				steps = append(steps, "step1")
				return nil
			},
			Compensate: func(ctx context.Context) error {
				steps = append(steps, "compensate1")
				return nil
			},
		})

		err := s.Execute(context.Background())
		assert.NoError(t, err)

		s.AddStep(saga.Step{
			Execute: func(ctx context.Context) error {
				steps = append(steps, "step2")
				return nil
			},
			Compensate: func(ctx context.Context) error {
				steps = append(steps, "compensate2")
				return nil
			},
		})

		assert.Equal(t, []string{"step1"}, steps, "Step should not be added after execution")
	})
}
