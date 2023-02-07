package queue_test

import (
	"fmt"
	"testing"

	"github.com/mycok/uSearch/bsp/queue"
)

func TestMsgEnqueueDequeueAndIteration(t *testing.T) {
	q := queue.NewInMemoryQueue()

	for i := 0; i < 10; i++ {
		err := q.Enqueue(msg{metadata: fmt.Sprint(i)})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	// Assert on pending messages.
	if !q.PendingMessages() {
		t.Error("Expected queue to have pending messages but got none")
	}

	// Assert that the messages are dequeued in reverse order.
	var (
		it             = q.Messages()
		numOfProcessed int
	)

	for msgIdx := 9; it.Next(); msgIdx-- {
		msgData := it.Message().(msg).metadata
		if msgData != fmt.Sprint(msgIdx) {
			t.Errorf("Expected %s, but got %s instead", msgData, fmt.Sprint(msgIdx))
		}

		numOfProcessed++
	}

	if numOfProcessed != 10 {
		t.Errorf(
			"Expected %d messages, but got %d messages instead",
			10,
			numOfProcessed,
		)
	}

	// Assert that the iterator didn't encounter any errors during iteration.
	if err := it.Error(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Discard pending messages.
	if err := q.DiscardMessages(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Assert on pending messages.
	if q.PendingMessages() {
		t.Error("Expected queue to 0 pending messages")
	}

	// Assert that the queue closes successfully.
	if err := q.Close(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// Message stab
type msg struct {
	metadata string
}

func (m msg) Type() string {
	return "msg is of type string"
}
