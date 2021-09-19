package detector

import (
	"net"

	"github.com/mycok/uSearch/cmd/crawler"
)

// Compile-time check for ensuring that Detector implements crawler.Detector.
var _ crawler.PrivateNetworkDetector = (*Detector)(nil)


var defaultPrivateCIDRs = []string{
	// Loopback.
	"127.0.0.0/8",
	"::1/128",
	// Private networks.
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	// Link-local addresses.
	"169.254.0.0/16",
	"fe80::/10",
	// Misc.
	"0.0.0.0/8", // All IP addresses on local machine.
	"255.255.255.255/32", // Broadcast address for the current network.
	"fc00::/7", // IPv6 unique local addr.
}

// Detector checks whether a host name resolves to a private network address.
type Detector struct {
	privateNetBlocks []*net.IPNet
}

// NewDetector returns a new Detector instance which is initialized with the
// default list of IPv4/IPv6 CIDR blocks that correspond to private networks
// according to RFC1918.
func New() (*Detector, error) {
	return NewDetectorFromCIDRs(defaultPrivateCIDRs...)
}

// NewDetectorFromCIDRs returns a new Detector instance which is initialized
// with the specified list of privateNetworkCIDRs.
func NewDetectorFromCIDRs(privateNetworkCIDRs ...string) (*Detector, error) {
	blocks, err := parseCIDRs(privateNetworkCIDRs)
	if err != nil {
		return nil, err
	}

	return &Detector{privateNetBlocks: blocks}, nil
}

// IsPrivate returns true if address resolves to a private network.
func (d *Detector) IsPrivate(address string) (bool, error) {
	ipAddr, err := net.ResolveIPAddr("ip", address)
	if err != nil {
		return false, err
	}

	for _, block := range d.privateNetBlocks {
		if block.Contains(ipAddr.IP) {
			return true, nil
		}
	}

	return false, nil
}

func parseCIDRs(cidrs []string) ([]*net.IPNet, error) {
	var (
		err error
		out = make([]*net.IPNet, len(cidrs))
	)

	for i, cidr := range cidrs {
		if _, out[i], err = net.ParseCIDR(cidr); err != nil {
			return nil, err
		}
	}

	return out, nil
}