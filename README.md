# Saga

Saga is a Go library that implements the saga pattern for distributed transactions.

## Installation

```
go get github.com/bhaweshksingh/saga-go
```

## Usage

Here's a basic example of how to use the Saga library:

```go
s := saga.New()

s.AddStep(saga.Step{
    Execute: func(ctx context.Context) error {
        // Perform some action
        return nil
    },
    Compensate: func(ctx context.Context) error {
        // Undo the action
        return nil
    },
})

// Add more steps...

err := s.Execute(context.Background())
if err != nil {
    // Handle the error
}
```

See the `example_test.go` file for more detailed usage examples.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

