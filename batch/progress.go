package batch

// EventType identifies the kind of event emitted during batch processing.
type EventType string

// Event types.
const (
	EventProgress EventType = "progress"
	EventItem     EventType = "item"
	EventComplete EventType = "complete"
	EventError    EventType = "error"
)

// Progress tracks the current state of a batch operation.
type Progress struct {
	Total     int
	Completed int
	Failed    int
	Status    string
}

// ProgressCallback is invoked with progress updates during batch processing.
type ProgressCallback func(Progress)

// Event represents a single event emitted during async batch processing.
type Event struct {
	Type     EventType
	Progress *Progress
	Result   *Result
	Err      error
}
