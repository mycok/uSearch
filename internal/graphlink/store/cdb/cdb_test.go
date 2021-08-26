package cdb

import (
	"database/sql"
	"os"
	"testing"

	"github.com/mycok/uSearch/internal/graphlink/graph/graphtests"

	check "gopkg.in/check.v1"
)

// Initialize and register an instance of the CockroachDBTestSuite to be
// executed by check testing package
var _ = check.Suite(new(CockroachDBTestSuite))

// CockroachDBTestSuite encapsulates the BaseSuite type tests methods
type CockroachDBTestSuite struct {
	db *sql.DB
	graphtests.BaseSuite
}

// Register our test suite with [go.test]
func Test(t *testing.T) {
	check.TestingT(t)
}

// SetUpSuite initializes and sets up the necessary testing env for the test suite
func (s *CockroachDBTestSuite) SetUpSuite(c *check.C) {
	dsn := os.Getenv("CDB_DSN")
	if dsn == "" {
		c.Skip("Missing CDB_DSN envvar: skipping cockroachDB backed test suite")
	}

	g, err := NewCockroachDbGraph(dsn)
	c.Assert(err, check.IsNil)
	s.SetGraph(g)

	// Keep track of the sql.DB instance so we can execute SQL statements to
	// reset the db between tests
	s.db = g.db
}

// TearDownSuite runs after the entire test suite has completed running. it
// resets the database and closes the db connection if open.
func (s *CockroachDBTestSuite) TearDownSuite(c *check.C) {
	if s.db != nil {
		s.flushDB(c)
		c.Assert(s.db.Close(), check.IsNil)
	}
} 

// SetUpTest runs before each test in the test suite. it's
// responsible for setting up the requirements necessary for
// running that specific test. ie database reset.
func (s *CockroachDBTestSuite) SetupTest(c *check.C) {
	s.flushDB(c)
}

// flushDB helper reset the database by deleting all link and
// edge entries from the links and edges tables.
func (s *CockroachDBTestSuite) flushDB(c *check.C) {
	_, err := s.db.Exec("TRUNCATE links CASCADE")
	c.Assert(err, check.IsNil)
}
