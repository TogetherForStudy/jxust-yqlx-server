package worker

import "context"

// TaskProcessor defines the interface for processing specific task types.
// Each processor handles one or more related task types (e.g., question tasks).
type TaskProcessor interface {
	// ProcessTask executes the task's business logic
	ProcessTask(ctx context.Context, task Task) error

	// Unmarshal deserializes task data from JSON bytes
	Unmarshal(data []byte) (Task, error)

	// GetSupportedTypes returns the list of task types this processor handles
	GetSupportedTypes() []string
}
