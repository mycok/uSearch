package es

import (
	"os"
	"strings"
	"testing"

	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/textindexer/index/indextest"
)

// Initialize and register an instance of the esIndexTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(esIndexTestSuite))

// Test registers the [check] library with the go testing library and enables
// the running of the test suite using the go testing library.
func Test(t *testing.T) {
	check.TestingT(t)
}

// esIndexTestSuite embeds and runs the BaseSuite tests methods.
type esIndexTestSuite struct {
	idx *ElasticsearchIndex
	indextest.BaseSuite
}

// SetUpSuite runs only once before all tests in the test suite. it's
// responsible for setting up required resources necessary for
// running the entire suite. ie database setup or reset.
func (s *esIndexTestSuite) SetUpSuite(c *check.C) {
	nodeList := os.Getenv("ES_NODES")
	if nodeList == "" {
		c.Skip("Missing ES_NODES envvar: skipping elasticsearch index test suite")
	}

	idx, err := NewEsIndexer(strings.Split(nodeList, ","), true)
	if err != nil {
		c.Fatal(err)
	}

	s.SetIndex(idx)
	s.idx = idx
}

// SetUpTest runs before each test in the test suite. it's
// responsible for setting up the necessary environment for
// running that specific test. ie database reset.
func (s *esIndexTestSuite) SetUpTest(c *check.C) {
	// Delete and create a new index / database table.
	if s.idx.client != nil {
		_, err := s.idx.client.Indices.Delete([]string{indexName})
		c.Assert(err, check.IsNil)

		err = initIndex(s.idx.client)
		c.Assert(err, check.IsNil)
	}
}

// TearDownSuite runs only once after all tests in the test suite. it's
// responsible for releasing all resources that were used to run the entire
// suite. ie dropping the database.
func (s *esIndexTestSuite) TearDownSuite(c *check.C) {
	if s.idx.client != nil {
		_, err := s.idx.client.Indices.Delete([]string{indexName})
		c.Assert(err, check.IsNil)
	}
}
