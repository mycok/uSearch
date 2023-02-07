package privnet_test

import (
	"testing"

	"github.com/mycok/uSearch/crawler/privnet"
)

func TestIpV4(t *testing.T) {
	testCases := []struct {
		description string
		input       string
		expected    bool
	}{
		{
			description: "loopback address",
			input:       "127.0.0.1",
			expected:    true,
		},
		{
			description: "private address (10.x.x.x)",
			input:       "10.0.0.128",
			expected:    true,
		},
		{
			description: "private address (192.x.x.x)",
			input:       "192.168.0.127",
			expected:    true,
		},
		{
			description: "private address (172.x.x.x)",
			input:       "172.16.10.10",
			expected:    true,
		},
		{
			description: "link-local address",
			input:       "169.254.169.254",
			expected:    true,
		},
	}

	netDetector, err := privnet.NewDetector()
	if err != nil {
		t.Fatal("Network detector initialization failed: ", err)
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			isPrivate, err := netDetector.IsNetworkPrivate(testCase.input)
			if err != nil {
				t.Error("Unexpected error: ", err)
			}

			if testCase.expected != isPrivate {
				t.Errorf(
					"Expected %q to be %v, got %v instead",
					testCase.description, testCase.expected, isPrivate,
				)
			}
		})
	}
}

func TestNetDetectorWithCustomCIDRs(t *testing.T) {
	// Create a new detector from a custom network address.
	netDetector, err := privnet.NewDetectorFromCIDRs("8.8.8.8/16")
	if err != nil {
		t.Fatal("Network detector initialization failed: ", err)
	}

	// Provide the IP address of the custom network address and check if it's
	// considered private.
	isPrivate, err := netDetector.IsNetworkPrivate("8.8.8.8")
	if err != nil {
		t.Error("Unexpected error: ", err)
	}

	if !isPrivate {
		t.Errorf(
			"Expected %q to be true, got %v instead",
			"8.8.8.8", isPrivate,
		)
	}
}
