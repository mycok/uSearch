package memory

import (
	"testing"

	"github.com/mycok/uSearch/internal/textindexer/index/indextests"

	check "gopkg.in/check.v1"
)

// Initialize and register an instance of the InMemoryBleveTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(InMemoryBleveTestSuite))

// InMemoryBleveTestSuite embeds the BaseSuite type tests methods.
type InMemoryBleveTestSuite struct {
	indextests.BaseSuite
	idx *InMemoryBleveIndexer
}

// Register our test suite with [go.test].
func Test(t *testing.T) {
	check.TestingT(t)
}

// SetUpTest runs before each test in the test suite. it's
// responsible for setting up the requirements necessary for
// running that specific test. ie database reset.
func (s *InMemoryBleveTestSuite) SetupTest(c *check.C) {
	idx, err := NewInMemoryBleveIndexer()
	c.Assert(err, check.IsNil)

	s.SetIndexer(idx)

	// Keep track of the concrete indexer implementation to be used for
	// clean up during the test tear down process.
	s.idx = idx
}

// TearDownTest ensures that the indexer instance connection is closed
// and all allocated resources are released.
func (s *InMemoryBleveTestSuite) TearDownTest(c *check.C) {	
	c.Assert(s.idx.Close(), check.IsNil)
}
