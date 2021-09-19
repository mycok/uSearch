package detector_test

import (
	"testing"

	"github.com/mycok/uSearch/cmd/crawler/detector"

	check "gopkg.in/check.v1"
)

// Initialize and register an instance of the DetectorTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(DetectorTestSuite))

type DetectorTestSuite struct {}

func Test(t *testing.T) {
	check.TestingT(t)
}

func (s *DetectorTestSuite) TestIpv4(c *check.C) {
	specs := []struct{
		desc string
		input string
		expected bool
	}{
		{
			desc: "loopback address",
			input: "127.0.0.1",
			expected:   true,
		},
		{
			desc: "private address (10.x.x.x)",
			input: "10.0.0.128",
			expected:   true,
		},
		{
			desc: "private address (192.x.x.x)",
			input: "192.168.0.127",
			expected:   true,
		},
		{
			desc: "private address (172.x.x.x)",
			input: "172.16.10.10",
			expected:   true,
		},
		{
			desc: "link-local address",
			input: "169.254.169.254",
			expected:   true,
		},
		{
			desc: "non-private address",
			input: "8.8.8.8",
			expected:   false,
		},
	}

	det, err := detector.New()
	c.Assert(err, check.IsNil)

	for specIndex, spec := range specs {
		c.Logf("[spec %d] %s:", specIndex, spec)
		private, err := det.IsPrivate(spec.input)

		c.Assert(err, check.IsNil)
		c.Assert(private, check.Equals, spec.expected)
	}
}

func (s *DetectorTestSuite) TestDetectorWithCustomCIDRs(c *check.C) {
	// Create a new detector from a custom network address.
	det, err := detector.NewDetectorFromCIDRs("8.8.8.8/16")
	c.Assert(err, check.IsNil)

	// Pass the IP address of the custom network address and check if it's
	// considered private. 
	private, err := det.IsPrivate("8.8.8.8")
	c.Assert(err, check.IsNil)
	c.Assert(private, check.Equals, true)
}