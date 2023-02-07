package memory

import (
	"testing"

	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/textindexer/index/indextest"
)

// Initialize and register a pointer instance of the inMemoryIndexTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(inMemoryIndexTestSuite))

// Test registers the [check] library with the go testing library and enables
// the running of the test suite using the go testing library.
func Test(t *testing.T) {
	check.TestingT(t)
}

// inMemoryIndexTestSuite embeds and runs the BaseSuite tests methods.
type inMemoryIndexTestSuite struct {
	idx *InMemoryIndex
	indextest.BaseSuite
}

// SetUpTest runs before each test in the test suite. it's
// responsible for setting up the necessary environment for
// running that specific test. ie database reset.
func (s *inMemoryIndexTestSuite) SetUpTest(c *check.C) {
	idx, err := NewInMemoryIndex()
	if err != nil {
		c.Fatalf("Failed to make a database connection: %v", err)
		c.Skip("Skipping Bleve backed test suite")
	}

	s.SetIndex(idx)

	// Keep track of the concrete index implementation to be used for
	// clean up during the test tear down process.
	s.idx = idx
}

// TearDownTest ensures that the index instance connection is closed
// and all allocated resources are released.
func (s *inMemoryIndexTestSuite) TearDownTest(c *check.C) {
	c.Assert(
		s.idx.Close(), check.IsNil,
		check.Commentf("Failed to close bleve connection"),
	)
}
