package message

import "sync"

// inMemoryQueue implements a queue that stores messages in memory. Messages
// can be enqueued concurrently but the returned iterator is not safe for
// concurrent access.
type inMemoryQueue struct {
	mu   sync.Mutex
	msgs []Message
	msg  Message
}

// NewInMemoryQueue creates a new in-memory queue instance. This function can
// serve as a QueueFactory.
func NewInMemoryQueue() Queue {
	return new(inMemoryQueue)
}

// Enqueue implements Queue.Enqueue.
func (q *inMemoryQueue) Enqueue(msg Message) error {
	q.mu.Lock()

	q.msgs = append(q.msgs, msg)

	q.mu.Unlock()

	return nil
}

// PendingMessages implements Queue.PendingMessages.
func (q *inMemoryQueue) PendingMessages() bool {
	q.mu.Lock()

	pending := len(q.msgs) != 0

	q.mu.Unlock()

	return pending
}

// DiscardMessages implements Queue.DiscardMessages.
func (q *inMemoryQueue) DiscardMessages() error {
	q.mu.Lock()

	q.msgs = q.msgs[:0]
	q.msg = nil

	q.mu.Unlock()

	return nil
}

// Close implements Queue.Close.
func (q *inMemoryQueue) Close() error {
	return nil
}

// Messages implements Queue.Messages.
func (q *inMemoryQueue) Messages() {
	return q
}

// Next implements Iterator.Next.
func (q *inMemoryQueue) Next() bool {
	q.mu.Lock()

	qMsgsLen := len(q.msgs)
	if qMsgsLen == 0 {
		q.mu.Unlock()
		return false
	}

	// Dequeue last message from the queue.
	q.msg = q.msgs[qMsgsLen-1]
	q.msgs = q.msgs[:qMsgsLen-1]

	q.mu.Unlock()

	return true
}

// Message implements Queue.Message.
func (q *inMemoryQueue) Message() Message {
	q.mu.Lock()

	msg := q.msg

	q.mu.Unlock()

	return msg
}

// Error implements Queue.Error.
func (*inMemoryQueue) Error() error {
	return nil
}
