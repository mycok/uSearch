package partition

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	// The following functions are overridden in tests.
	getHostname = os.Hostname
	// Lookup a service on a network.
	lookupSRV = net.LookupSRV
	// ErrNoPartitionDataAvailableYet is returned by the SRV-aware
	// partition detector to indicate that SRV records for this target
	// application are not yet available. SRV(service) record creation can
	// sometimes take some bit of time after a stateful set has been deployed.
	ErrNoPartitionDataAvailableYet = errors.New("no partition data available yet")
)

// Detector should be implemented by types that assign an application instance in a cluster
// to a particular data source partition. [ie link data store partitions].
type Detector interface {
	// PartitionInfo extracts and returns the current partition number from the current
	// host_name along with the total number of partitions.
	PartitionInfo() (int, int, error)
}

// SRVRecord detects the number of partitions by performing a SRV (service) query
// and counting the number of results.
type SRVRecord struct {
	// Headless service name.
	srvName string
}

// DetectFromSRVRecords returns a PartitionDetector implementation that
// extracts the current partition name from the current host name and attempts
// to detect the total number of partitions by performing a service [SRV] query
// and counting the number of responses.
//
// This detector is meant to be used in conjunction with a Stateful Set in
// a kubernetes environment.
func DetectFromSRVRecords(srvName string) SRVRecord {
	return SRVRecord{srvName: srvName}
}

// PartitionInfo extracts and returns the current partition number from the current
// host_name along with the total number of partitions which it attempts to detect
// by performing a SRV (service) query and counting the number of responses.
func (det SRVRecord) PartitionInfo() (int, int, error) {
	// This query will return the hostname of the pod hosting the app / service
	// with the format [SERVICE_NAME-INDEX]. INDEX represents the position of
	// the pod in the entire stateful set. We use that INDEX as a partition number.
	hostname, err := getHostname()
	if err != nil {
		return -1, -1, fmt.Errorf("partition detector: unable to detect host name: %w", err)
	}

	// Extract the index part of the hostname.
	tokens := strings.Split(hostname, "-")
	// Convert the index into a 32-bit integer that represents a partition number..
	partition, err := strconv.ParseInt(tokens[len(tokens)-1], 10, 32)
	if err != nil {
		return -1, -1, errors.New(
			"partition detector: unable to extract partition number from the host name suffix",
		)
	}

	_, addrs, err := lookupSRV("", "", det.srvName)
	if err != nil {
		return -1, -1, ErrNoPartitionDataAvailableYet
	}

	return int(partition), len(addrs), nil
}

// DummyDetector is a partition detector implementation that always returns
// the same partition details.
type DummyDetector struct {
	Partition       int
	NumOfPartitions int
}

// PartitionInfo extracts and returns the current partition number from the current
// host_name along with the total number of partitions.
func (det DummyDetector) PartitionInfo() (int, int, error) {
	return det.Partition, det.NumOfPartitions, nil
}
