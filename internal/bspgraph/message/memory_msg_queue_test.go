package message_test

import (
	"fmt"
	"testing"

	"github.com/mycok/uSearch/internal/bspgraph/message"

	check "gopkg.in/check.v1"
)


// Initialize and register an instance of InMemoryMsgQueueTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(InMemoryMsgQueueTestSuite))

func Test(t *testing.T) {
	check.TestingT(t)
}

type InMemoryMsgQueueTestSuite struct {
	q message.Queue
}

func (s *InMemoryMsgQueueTestSuite) SetUpTest(c *check.C) {
	s.q = message.NewInMemoryQueue()
}

func (s *InMemoryMsgQueueTestSuite) TearDownTest(c *check.C) {
	c.Assert(s.q.Close(), check.IsNil)
}

func (s *InMemoryMsgQueueTestSuite) TestMsgEnqueueAndDequeue(c *check.C) {
	for i := 0; i < 10; i++ {
		err := s.q.Enqueue(msg{metaData: fmt.Sprint(i)})
		c.Assert(err, check.IsNil)
	}

	c.Assert(s.q.PendingMessages(), check.Equals, true)

	// We expect the messages to be dequeued in reverse order.
	var (
		it = s.q.Messages()
		numofProcessed int
	)

	for expNext := 9; it.Next(); expNext-- {
		result := it.Message().(msg).metaData
		c.Assert(result, check.Equals, fmt.Sprint(expNext))
		numofProcessed++
	}

	c.Assert(numofProcessed, check.Equals, 10)
	c.Assert(it.Error(), check.IsNil)
}

func (s *InMemoryMsgQueueTestSuite) TestMsgDiscard(c *check.C) {
	for i := 0; i < 10; i++ {
		err := s.q.Enqueue(msg{metaData: fmt.Sprint(i)})
		c.Assert(err, check.IsNil)
	}

	c.Assert(s.q.PendingMessages(), check.Equals, true)
	c.Assert(s.q.DiscardMessages(), check.IsNil)
	c.Assert(s.q.PendingMessages(), check.Equals, false)
}

type msg struct {
	metaData string
}

func (msg) Type() string {
	return "msg is of type string"
}
