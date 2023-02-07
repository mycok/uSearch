package queue

// Message should be implemented by types that can serve as message objects.
type Message interface {
	// Type returns the type of this Message.
	Type() string
}

// Queue should be implemented by types that can serve as message queues.
type Queue interface {
	// Enqueue adds a new message at the end of the queue.
	Enqueue(msg Message) error

	// PendingMessages checks the queue for unconsumed messages.
	PendingMessages() bool

	// DiscardMessages drops all unconsumed messages from the queue.
	DiscardMessages() error

	// Messages returns an iterator of queued messages.
	Messages() Iterator

	// Close releases all resources consumed by the queue.
	Close() error
}

// Iterator should be embedded / implemented by types that require
// iteration functionality.
type Iterator interface {
	// Next loads the next item, returns false when no more items
	// are available or when an error occurs.
	Next() bool

	// Message returns the current message from the result set.
	Message() Message

	// Error returns the last error encountered by the iterator.
	Error() error
}

// Factory creates new Queue instances.
// Note: Should be used for cases where lazy object creation is desired.
type Factory func() Queue
