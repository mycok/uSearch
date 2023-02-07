package privnet

import (
	"net"

	"github.com/mycok/uSearch/crawler"
)

// Static and compile-time check to ensure NetDetector implements
// crawler.PrivateNetworkDetector interface.
var _ crawler.PrivateNetworkDetector = (*NetDetector)(nil)

var defaultPrivateCIDRs = []string{
	// Loopback / Localhost.
	"127.0.0.0/8", // IPv4
	"::1/128",     // IPv6
	// Private networks.
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	// Link-local addresses.
	"169.254.0.0/16",
	"fe80::/10",
	// Misc.
	"0.0.0.0/8",          // All IP addresses on a local machine.
	"255.255.255.255/32", // Broadcast address for the current network.
	"fc00::/7",           // IPv6 unique local addr.
}

// NetDetector checks whether a host name resolves to a private network address.
type NetDetector struct {
	privateNetBlocks []*net.IPNet
}

// NewDetector returns a pointer to a NetDetector instance configured with a default
// list of IPv4/IPv6 CIDR blocks that correspond to private networks
// according to RFC1918.
func NewDetector() (*NetDetector, error) {
	return NewDetectorFromCIDRs(defaultPrivateCIDRs...)
}

// NewDetectorFromCIDRs returns a new Detector instance which is initialized
// with the specified list of privateNetworkCIDRs.
// Note: This serves as an alternative NetDetector constructor. it should be
// only when a client wants to provide their own list of private networks to
// be used by the NetDetector.
func NewDetectorFromCIDRs(privateNetworkCIDRs ...string) (*NetDetector, error) {
	netBlocks, err := parseCIDRs(privateNetworkCIDRs...)
	if err != nil {
		return nil, err
	}

	return &NetDetector{privateNetBlocks: netBlocks}, nil
}

// IsNetworkPrivate checks if the an address is part of a private network.
func (d *NetDetector) IsNetworkPrivate(address string) (bool, error) {
	ipAddr, err := net.ResolveIPAddr("ip", address)
	if err != nil {
		return false, err
	}

	for _, netBlock := range d.privateNetBlocks {
		if netBlock.Contains(ipAddr.IP) {
			return true, nil
		}
	}

	return false, nil
}

// parseCIDRs transforms network host strings into valid CIDR notation
// IP addresses.
func parseCIDRs(cidrs ...string) ([]*net.IPNet, error) {
	var err error
	ipNets := make([]*net.IPNet, len(cidrs))

	for i, host := range cidrs {
		if _, ipNets[i], err = net.ParseCIDR(host); err != nil {
			return nil, err
		}
	}

	return ipNets, nil
}
