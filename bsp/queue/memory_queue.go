package queue

import "sync"

// Static and compile-time check to ensure inMemoryQueue implements
// Queue interface.
var _ Queue = (*inMemoryQueue)(nil)

// inMemoryQueue stores messages in memory. Messages can be enqueued
// concurrently but the returned iterator is not safe for concurrent access.
type inMemoryQueue struct {
	mu   sync.Mutex
	msgs []Message
	msg  Message
}

// NewInMemoryQueue creates a new in-memory queue instance. This function can
// serve as a QueueFactory.
func NewInMemoryQueue() Queue {
	return &inMemoryQueue{}
}

// Enqueue adds a new message at the end of the queue.
func (q *inMemoryQueue) Enqueue(msg Message) error {
	q.mu.Lock()

	q.msgs = append(q.msgs, msg)

	q.mu.Unlock()

	return nil
}

// PendingMessages checks the queue for unconsumed messages.
func (q *inMemoryQueue) PendingMessages() bool {
	q.mu.Lock()

	pending := len(q.msgs) != 0

	q.mu.Unlock()

	return pending
}

// DiscardMessages drops all unconsumed messages from the queue.
func (q *inMemoryQueue) DiscardMessages() error {
	q.mu.Lock()

	q.msgs = q.msgs[:0]
	q.msg = nil

	q.mu.Unlock()

	return nil
}

// Messages returns an iterator of queued messages.
func (q *inMemoryQueue) Messages() Iterator {
	return q
}

// Close releases all resources consumed by the queue.
func (q *inMemoryQueue) Close() error {
	return nil
}

// Next loads the next item, returns false when no more items
// are available or when an error occurs.
func (q *inMemoryQueue) Next() bool {
	q.mu.Lock()

	size := len(q.msgs)
	if size == 0 {
		q.mu.Unlock()

		return false
	}

	// Dequeue last message from the queue. This strategy is intentional as
	// it doesn't reduce the original slice's capacity which is useful since
	// the queue object resets and re-use the same slice when discarding
	// unconsumed messages.
	q.msg = q.msgs[size-1]
	q.msgs = q.msgs[0 : size-1]

	q.mu.Unlock()

	return true
}

// Message returns the current message from the result set.
func (q *inMemoryQueue) Message() Message {
	q.mu.Lock()

	msg := q.msg

	q.mu.Unlock()

	return msg
}

// Error returns the last error encountered by the iterator.
func (q *inMemoryQueue) Error() error {
	return nil
}
