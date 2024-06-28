package saga

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Saga represents a series of steps that can be executed and compensated.
type Saga interface {
	// Execute runs all steps in the saga. If any step fails, it triggers compensation.
	Execute(ctx context.Context) error
	// AddStep adds a new step to the saga.
	AddStep(step Step)

	// Abort stops the saga execution and triggers compensation for completed steps.
	Abort(ctx context.Context) error
}

// Step represents a single step in a saga, including its execution and compensation logic.
type Step struct {
	// Execute is the main logic for this step.
	Execute func(ctx context.Context) error
	// Compensate is called to undo this step if a later step fails.
	Compensate func(ctx context.Context) error
}

type sagaState int

// Saga states
const (
	stateInitial sagaState = iota
	stateFinishing
	stateAborting
	stateFinished
)

// sagaImpl is the internal implementation of the Saga interface.
type sagaImpl struct {
	mu             sync.Mutex
	state          sagaState
	steps          []Step
	completedSteps []int
}

// New creates a new Saga instance.
func New() (Saga, func(context.Context, *error)) {
	s := sagaImpl{
		steps:          []Step{},
		completedSteps: []int{},
	}

	abort := func(ctx context.Context, err *error) {
		if abortErr := s.Abort(ctx); abortErr != nil {
			*err = fmt.Errorf("transaction rollback failed: %v, underlying error: %v", abortErr, *err)
		}
	}
	return &s, abort
}

// Abort stops the saga execution and triggers compensation for completed steps.
func (s *sagaImpl) Abort(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == stateFinished || s.state == stateAborting {
		return nil
	}
	s.state = stateAborting

	// If the saga is already finished or compensation is in progress, do nothing
	if len(s.completedSteps) == 0 {
		return nil
	}

	var compensationErrs []string
	contextCancelled := false

	for i := len(s.completedSteps) - 1; i >= 0; i-- {
		stepIndex := s.completedSteps[i]

		select {
		case <-ctx.Done():
			compensationErrs = append(compensationErrs, fmt.Sprintf("context cancelled during abort: %v", ctx.Err()))
			contextCancelled = true
		default:
			if err := s.steps[stepIndex].Compensate(ctx); err != nil {
				compensationErrs = append(compensationErrs, fmt.Sprintf("step %d: %v", stepIndex, err))
			}
		}

		if contextCancelled {
			break
		}
	}

	s.completedSteps = nil // Reset completed steps

	if len(compensationErrs) > 0 {
		return &SagaError{
			OriginalError:     fmt.Errorf("saga aborted"),
			CompensationError: fmt.Errorf("abort errors: [%s]", strings.Join(compensationErrs, "; ")),
		}
	}

	return nil
}

// Execute implements the Saga interface.
func (s *sagaImpl) Execute(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != stateInitial {
		return nil
	}
	s.state = stateFinishing

	for i, step := range s.steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := step.Execute(ctx); err != nil {
				return s.compensate(ctx, i, err)
			}
			s.completedSteps = append(s.completedSteps, i)
		}
	}

	return nil
}

// AddStep implements the Saga interface.
func (s *sagaImpl) AddStep(step Step) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == stateInitial {
		s.steps = append(s.steps, step)
	}
}

// compensate runs compensation logic for all completed steps in reverse order.
func (s *sagaImpl) compensate(ctx context.Context, failedStep int, origErr error) error {
	var compensationErrs []string

	for i := len(s.completedSteps) - 1; i >= 0; i-- {
		stepIndex := s.completedSteps[i]
		if stepIndex >= failedStep {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.steps[stepIndex].Compensate(ctx); err != nil {
				compensationErrs = append(compensationErrs, fmt.Sprintf("step %d: %v", stepIndex, err))
			}
		}
	}

	if len(compensationErrs) > 0 {
		return &SagaError{
			OriginalError:     origErr,
			CompensationError: fmt.Errorf("compensation errors: [%s]", strings.Join(compensationErrs, "; ")),
		}
	}

	return origErr
}

// SagaError represents an error that occurred during saga execution, including any compensation errors.
type SagaError struct {
	OriginalError     error
	CompensationError error
}

func (e *SagaError) Error() string {
	return fmt.Sprintf("saga failed: %v; compensation errors: %v", e.OriginalError, e.CompensationError)
}

// File: example_test.go
