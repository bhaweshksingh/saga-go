package saga_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bhaweshksingh/saga-go"
	"github.com/stretchr/testify/assert"
)

func TestSaga(t *testing.T) {
	t.Run("Successful execution", func(t *testing.T) {
		s := saga.New()
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
		s := saga.New()
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
		s := saga.New()
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
		assert.Equal(t, []string{"step1", "step2", "compensate1"}, steps)
		assert.Contains(t, err.Error(), "step 2 failed")
		assert.Contains(t, err.Error(), "compensation 1 failed")
	})

	t.Run("Context cancellation", func(t *testing.T) {
		s := saga.New()
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
				time.Sleep(100 * time.Millisecond)
				steps = append(steps, "step2")
				return nil
			},
			Compensate: func(ctx context.Context) error {
				steps = append(steps, "compensate2")
				return nil
			},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := s.Execute(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
		assert.Equal(t, []string{"step1", "compensate1"}, steps)
	})
}
