package memory

type (
	// Execute defines the method to execute the task.
	Execute func(key, value interface{})
)
