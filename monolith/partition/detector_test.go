package partition

import (
	"errors"
	"net"
	"os"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(new(DetectorTestSuite))

type DetectorTestSuite struct{}

func (s *DetectorTestSuite) SetUpTest(c *check.C) {
	getHostname = os.Hostname
	lookupSRV = net.LookupSRV
}

func (s *DetectorTestSuite) TearDownTest(c *check.C) {
	getHostname = os.Hostname
	lookupSRV = net.LookupSRV
}

func (s *DetectorTestSuite) TestDetectFromSRVRecords(c *check.C) {
	getHostname = func() (string, error) {
		return "web-1", nil
	}

	lookupSRV = func(service, proto, name string) (cname string, addrs []*net.SRV, err error) {
		c.Assert(service, check.Equals, "")
		c.Assert(proto, check.Equals, "")
		c.Assert(name, check.Equals, "web-service")

		return "web-service", make([]*net.SRV, 4), nil
	}

	det := DetectFromSRVRecords("web-service")
	currPartition, numOfPartitions, err := det.PartitionInfo()

	c.Assert(err, check.IsNil)
	c.Assert(currPartition, check.Equals, 1)
	c.Assert(numOfPartitions, check.Equals, 4)
}

func (s *DetectorTestSuite) TestDetectFromSRVRecordsWithNoAvailableData(c *check.C) {
	getHostname = func() (string, error) {
		return "web-1", nil
	}

	lookupSRV = func(service, proto, name string) (cname string, addrs []*net.SRV, err error) {
		return "", nil, errors.New("host not found")
	}

	det := DetectFromSRVRecords("web-service")
	_, _, err := det.PartitionInfo()
	c.Assert(errors.Is(err, ErrNoPartitionDataAvailableYet), check.Equals, true)
}
