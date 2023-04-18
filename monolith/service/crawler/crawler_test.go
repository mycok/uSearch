package crawler

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/juju/clock/testclock"
	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/monolith/partition"
	"github.com/mycok/uSearch/monolith/service/crawler/mocks"
)

var _ = check.Suite(new(ConfigTestSuite))
var _ = check.Suite(new(CrawlerServiceTestSuite))

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
		NumOfFetchWorkers:   4,
		CrawlUpdateInterval: time.Minute,
		ReIndexThreshold:    time.Minute,
	}

	config := originalConfig
	c.Assert(config.validate(), check.IsNil)
	c.Assert(config.PrivateNetworkDetector, check.Not(check.IsNil), check.Commentf("default private network detector was not assigned"))
	c.Assert(config.URLGetter, check.Not(check.IsNil), check.Commentf("default URL getter was not assigned"))
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
	config.NumOfFetchWorkers = 0
	c.Assert(config.validate(), check.ErrorMatches, "(?ms).*invalid value for fetch workers.*")

	config = originalConfig
	config.CrawlUpdateInterval = 0
	c.Assert(config.validate(), check.ErrorMatches, "(?ms).*invalid value for crawl update interval interval.*")

	config = originalConfig
	config.ReIndexThreshold = 0
	c.Assert(config.validate(), check.ErrorMatches, "(?ms).*invalid value for re-index threshold.*")
}

type CrawlerServiceTestSuite struct{}

func (s *CrawlerServiceTestSuite) TestFullRun(c *check.C) {
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
		NumOfFetchWorkers:   1,
		CrawlUpdateInterval: time.Minute,
		ReIndexThreshold:    12 * time.Hour,
	}

	svc, err := New(config)
	c.Assert(err, check.IsNil)

	ctx, cancelFn := context.WithCancel(context.TODO())
	defer cancelFn()

	mockIt := mocks.NewMockLinkIterator(ctrl)
	mockIt.EXPECT().Next().Return(false)
	mockIt.EXPECT().Error().Return(nil)
	mockIt.EXPECT().Close().Return(nil)

	expectedLinkFilterTime := clk.Now().Add(config.CrawlUpdateInterval).Add(-config.ReIndexThreshold)
	mockGraph.EXPECT().Links(
		uuid.Nil,
		uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		expectedLinkFilterTime,
	).Return(mockIt, nil)

	go func() {
		// Wait until the main loop calls time.After (or timeout if 10
		// sec elapse) and advance the time to trigger a new crawler
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
