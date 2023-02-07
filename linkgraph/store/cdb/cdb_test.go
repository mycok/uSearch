package cdb

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/linkgraph/graph/graphtest"
)

// Initialize and register an instance of the cockroachDBGraphTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(cockroachDBGraphTestSuite))

// Test registers the [check] library with the go testing library and enables
// the running of the test suite using the go testing library.
func Test(t *testing.T) {
	check.TestingT(t)
}

// cockroachDBGraphTestSuite embeds and runs the BaseSuite tests methods.
type cockroachDBGraphTestSuite struct {
	// Keep track of the sql.DB instance from the graph implementation
	// so we can execute SQL statements to reset the db between tests.
	db *sql.DB
	graphtest.BaseSuite
}

// SetUpSuite runs only once before all tests in the test suite. it's
// responsible for setting up required resources necessary for
// running the entire suite. ie database setup or reset.
func (s *cockroachDBGraphTestSuite) SetUpSuite(c *check.C) {
	dsn := os.Getenv("CDB_DSN")
	if dsn == "" {
		c.Skip("Missing CDB_DSN envvar: skipping cockroachDB backed test suite")
	}

	g, err := NewCockroachDBGraph(dsn)
	if err != nil {
		c.Fatalf("Failed to make a database connection: %v", err)
		c.Skip("Missing CDB_DSN envvar: skipping cockroachDB backed test suite")
	}

	s.SetGraph(g)
	// Pass graph db instance reference forward to the suite,
	s.db = g.db
}

// TearDownSuite runs only once after the entire test suite has completed
// running. it resets the database and closes the db connection if open.
func (s *cockroachDBGraphTestSuite) TearDownSuite(c *check.C) {
	if s.db != nil {
		s.flushDB(c)
		c.Assert(s.db.Close(), check.IsNil)
	}
}

// SetUpTest runs before each test in the test suite. it's
// responsible for setting up the necessary environment for
// running that specific test. ie database reset.
func (s *cockroachDBGraphTestSuite) SetUpTest(c *check.C) {
	s.flushDB(c)
}

// flushDB helper resets the database by deleting all link and
// edge entries from the links and edges tables.
func (s *cockroachDBGraphTestSuite) flushDB(c *check.C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := s.db.ExecContext(ctx, "TRUNCATE links CASCADE")
	c.Assert(err, check.IsNil)
}
