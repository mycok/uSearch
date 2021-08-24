package memory

import (
	"testing"

	"github.com/mycok/uSearch/internal/graphlink/graph/graphtests"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(new(InMemoryGraphTestSuite))

type InMemoryGraphTestSuite struct {
	graphtests.BaseSuite
}

func Test(t *testing.T) {
	check.TestingT(t)
}

func (s *InMemoryGraphTestSuite) SetUpTest(c *check.C) {
	s.SetGraph(NewInMemoryGraph())
}
