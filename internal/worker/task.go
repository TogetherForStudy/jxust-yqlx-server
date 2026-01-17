package worker

import "time"

// Task represents a unit of work that can be queued and processed asynchronously.
// Implementations must be JSON-serializable and contain all data needed for processing.
type Task interface {
	// GetType returns the task type identifier (e.g., "study", "practice", "usage")
	GetType() string

	// Marshal serializes the task to JSON bytes for queue storage
	Marshal() ([]byte, error)

	// GetRetryCount returns the current retry attempt count
	GetRetryCount() int

	// IncrementRetry increments the retry counter
	IncrementRetry()

	// GetTimestamp returns when the task was created
	GetTimestamp() time.Time
}
