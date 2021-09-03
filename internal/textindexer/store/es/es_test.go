package es

import (
	"testing"
	"os"
	"strings"

	"github.com/mycok/uSearch/internal/textindexer/index/indextests"

	check "gopkg.in/check.v1"
)

// Initialize and register a pointer instance of the ElasticsearchTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(ElasticsearchTestSuite))

// ElasticsearchTestSuite embeds the BaseSuite type tests methods.
type ElasticsearchTestSuite struct {
	indextests.BaseSuite
	idx *ElasticsearchIndexer
}

// Register test suite with [go.test].
func Test(t *testing.T) {
	check.TestingT(t)
}

// SetUpSuite initializes and sets up the necessary testing env for the test suite.
func (s *ElasticsearchTestSuite) SetUpSuite(c *check.C) {
	nodeList := os.Getenv("ES_NODES")
	if nodeList == "" {
		c.Skip("Missing ES_NODES envvar: skipping elasticsearch index test suite")
	}

	idx, err := NewElasticsearchIndexer(strings.Split(nodeList, ","), true)
	c.Assert(err, check.IsNil)

	s.SetIndexer(idx)
	s.idx = idx
}

func (s *ElasticsearchTestSuite) SetUpTest(c *check.C) {
	if s.idx.es != nil {
		_, err := s.idx.es.Indices.Delete([]string{indexName})
		c.Assert(err, check.IsNil)

		err = createIndex(s.idx.es)
		c.Assert(err, check.IsNil)
	}
}

