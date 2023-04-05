package service

import (
	"context"
	"testing"
	"fmt"
	"time"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(new(GroupTestSuite))

func Test(t *testing.T) {
	check.TestingT(t)
}

type GroupTestSuite struct {}

func (s *GroupTestSuite) TestServiceGroupTerminatesAfterASingleError(c *check.C) {
	grp := Group{
		testService{id: "0"},
		testService{id: "1", err: fmt.Errorf("failed to connect to API")},
		testService{id: "2"},
	}

	err := grp.Execute(context.TODO())
	c.Assert(err, check.Not(check.IsNil))
	c.Assert(err, check.ErrorMatches, "(?ms).*1: failed to connect to API.*")
}

func (s *GroupTestSuite) TestServiceGroupTerminatesAfterMultipleErrors(c *check.C) {
	grp := Group{
		testService{id: "0"},
		testService{id: "1", err: fmt.Errorf("failed to connect to API")},
		testService{id: "2", err: fmt.Errorf("failed to connect to API")},
	}

	err := grp.Execute(context.TODO())
	c.Assert(err, check.Not(check.IsNil))
	c.Assert(err, check.ErrorMatches, "(?ms).*1: failed to connect to API.*")
	c.Assert(err, check.ErrorMatches, "(?ms).*1: failed to connect to API.*")
}

func (s *GroupTestSuite) TestServiceGroupTerminatesFromContext(c *check.C) {
	grp := Group{
		testService{id: "0"},
		testService{id: "1"},
		testService{id: "2"},
	}

	ctx, cancelFn := context.WithTimeout(context.TODO(), 200*time.Millisecond)
	defer cancelFn()
	err := grp.Execute(ctx)
	c.Assert(err, check.IsNil)
}

type testService struct {
	id string
	err error
}
 func (s testService) Name() string { return s.id }
 func (s testService) Run(ctx context.Context) error {
 	if s.err != nil {
 		return s.err
 	}

 	<-ctx.Done()

 	return nil
 }