package pagerank

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/juju/clock/testclock"
	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/monolith/partition"
	"github.com/mycok/uSearch/monolith/service/pagerank/mocks"
)

var _ = check.Suite(new(ConfigTestSuite))
var _ = check.Suite(new(PageRankServiceTestSuite))

func Test(t *testing.T) {
	check.TestingT(t)
}

type ConfigTestSuite struct{}

func (s *ConfigTestSuite) TestConfigValidation(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	originalConfig := Config{
		GraphAPI:            mocks.NewMockGraphAPI(ctrl),
		IndexAPI:            mocks.NewMockIndexAPI(ctrl),
		PartitionDetector:   partition.DummyDetector{},
		NumOfComputeWorkers: 4,
		UpdateInterval:      time.Minute,
	}

	config := originalConfig
	c.Assert(config.validate(), check.IsNil)

	c.Assert(config.Clock, check.Not(check.IsNil), check.Commentf("default clock was not assigned"))
	c.Assert(config.Logger, check.Not(check.IsNil), check.Commentf("default logger was not assigned"))

	config = originalConfig
	config.GraphAPI = nil
	c.Assert(config.validate(), check.ErrorMatches, "(?ms).*graph API not provided.*")

	config = originalConfig
	config.IndexAPI = nil
	c.Assert(config.validate(), check.ErrorMatches, "(?ms).*index API not provided.*")

	config = originalConfig
	config.PartitionDetector = nil
	c.Assert(config.validate(), check.ErrorMatches, "(?ms).*partition detector not provided.*")

	config = originalConfig
	config.NumOfComputeWorkers = 0
	c.Assert(config.validate(), check.ErrorMatches, "(?ms).*invalid value for compute workers.*")

	config = originalConfig
	config.UpdateInterval = 0
	c.Assert(config.validate(), check.ErrorMatches, "(?ms).*invalid value for update interval interval.*")
}

type PageRankServiceTestSuite struct{}

func (s *PageRankServiceTestSuite) TestFullRun(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	mockGraph := mocks.NewMockGraphAPI(ctrl)
	mockIndex := mocks.NewMockIndexAPI(ctrl)
	clk := testclock.NewClock(time.Now())

	config := Config{
		GraphAPI:            mockGraph,
		IndexAPI:            mockIndex,
		PartitionDetector:   partition.DummyDetector{Partition: 0, NumOfPartitions: 1},
		Clock:               clk,
		NumOfComputeWorkers: 1,
		UpdateInterval:      time.Minute,
	}

	svc, err := New(config)
	c.Assert(err, check.IsNil)

	ctx, cancelFn := context.WithCancel(context.TODO())
	defer cancelFn()

	uuid1, uuid2 := uuid.New(), uuid.New()

	mockLinkIt := mocks.NewMockLinkIterator(ctrl)
	gomock.InOrder(
		mockLinkIt.EXPECT().Next().Return(true),
		mockLinkIt.EXPECT().Link().Return(&graph.Link{ID: uuid1}),
		mockLinkIt.EXPECT().Next().Return(true),
		mockLinkIt.EXPECT().Link().Return(&graph.Link{ID: uuid2}),
		mockLinkIt.EXPECT().Next().Return(false),
	)
	mockLinkIt.EXPECT().Error().Return(nil)
	mockLinkIt.EXPECT().Close().Return(nil)

	mockEdgeIt := mocks.NewMockEdgeIterator(ctrl)
	gomock.InOrder(
		mockEdgeIt.EXPECT().Next().Return(true),
		mockEdgeIt.EXPECT().Edge().Return(&graph.Edge{Src: uuid1, Dest: uuid2}),
		mockEdgeIt.EXPECT().Next().Return(true),
		mockEdgeIt.EXPECT().Edge().Return(&graph.Edge{Src: uuid2, Dest: uuid1}),
		mockEdgeIt.EXPECT().Next().Return(false),
	)
	mockEdgeIt.EXPECT().Error().Return(nil)
	mockEdgeIt.EXPECT().Close().Return(nil)

	expectedLinkFilterTime := clk.Now().Add(config.UpdateInterval)
	maxUUID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

	mockGraph.EXPECT().Links(
		uuid.Nil, maxUUID, expectedLinkFilterTime,
	).Return(mockLinkIt, nil)
	mockGraph.EXPECT().Edges(
		uuid.Nil, maxUUID, expectedLinkFilterTime,
	).Return(mockEdgeIt, nil)

	mockIndex.EXPECT().UpdateScore(uuid1, 0.5)
	mockIndex.EXPECT().UpdateScore(uuid2, 0.5)

	go func() {
		// Wait until the main loop calls time.After (or timeout if 10
		// sec elapse) and advance the time to trigger a new page rank
		// pass.
		c.Assert(clk.WaitAdvance(time.Minute, 10*time.Second, 1), check.IsNil)

		// Wait until the main loop calls time.After again and cancel
		// the context.
		c.Assert(clk.WaitAdvance(time.Millisecond, 10*time.Second, 1), check.IsNil)
		cancelFn()
	}()

	// Enter the blocking main loop.
	err = svc.Run(ctx)
	c.Assert(err, check.IsNil)
}

func (s *PageRankServiceTestSuite) TestRunOnNonMasterPartition(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	clk := testclock.NewClock(time.Now())

	config := Config{
		GraphAPI:            mocks.NewMockGraphAPI(ctrl),
		IndexAPI:            mocks.NewMockIndexAPI(ctrl),
		PartitionDetector:   partition.DummyDetector{Partition: 1, NumOfPartitions: 2},
		Clock:               clk,
		NumOfComputeWorkers: 1,
		UpdateInterval:      time.Minute,
	}

	svc, err := New(config)
	c.Assert(err, check.IsNil)

	go func() {
		// Wait until the main loop calls time.After and advance the time.
		// The service will check the partition information, see that
		// it is not assigned to partition 0 and exit the main loop.
		c.Assert(clk.WaitAdvance(time.Minute, 10*time.Second, 1), check.IsNil)
	}()

	// Enter the blocking main loop
	err = svc.Run(context.TODO())
	c.Assert(err, check.IsNil)
}
