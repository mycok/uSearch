package partition

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

// Range represents a contiguous UUID region which is split into a number of
// partitions.
type Range struct {
	start      uuid.UUID
	partitions []uuid.UUID // Partitions carved from a particular UUID range.
}

// NewFullRange creates a new range that uses the full UUID value space and
// splits it into the provided number of partitions.
func NewFullRange(numOfPartitions int) (Range, error) {
	return NewRange(
		numOfPartitions,
		uuid.Nil,
		uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
	)
}

// NewRange creates a new range [start, end] and splits it into the
// provided number of partitions.
func NewRange(numOfPartitions int, start, end uuid.UUID) (Range, error) {
	// Check if the start is greater than or equal to end.
	if bytes.Compare(start[:], end[:]) >= 0 {
		return Range{}, errors.New(
			"range start UUID must be less than the end UUID",
		)
	}

	if numOfPartitions <= 0 {
		return Range{}, errors.New(
			"number of partitions must be at least equal to 1",
		)
	}

	// Calculate the size of each partition as:
	// ((end - start + 1) / numPartitions).
	tokenRange := big.NewInt(0)
	partitionSize := big.NewInt(0)
	partitionSize = partitionSize.Sub(
		big.NewInt(0).SetBytes(end[:]),
		big.NewInt(0).SetBytes(start[:]),
	)
	partitionSize = partitionSize.Div(
		partitionSize.Add(partitionSize, big.NewInt(1)),
		big.NewInt(int64(numOfPartitions)),
	)

	var (
		to         uuid.UUID
		err        error
		partitions = make([]uuid.UUID, numOfPartitions)
	)

	for partition := 0; partition < numOfPartitions; partition++ {
		// If it's the last partition.
		if partition == numOfPartitions-1 {
			to = end
		} else {
			tokenRange.Mul(partitionSize, big.NewInt(int64(partition+1)))
			if to, err = uuid.FromBytes(tokenRange.Bytes()); err != nil {
				return Range{}, fmt.Errorf("partition range: %w", err)
			}
		}

		partitions[partition] = to
	}

	return Range{
		start:      start,
		partitions: partitions,
	}, nil
}

// PartitionRange returns the [start, end] range for the requested partition.
func (r Range) PartitionRange(partition int) (uuid.UUID, uuid.UUID, error) {
	if partition < 0 || partition >= len(r.partitions) {
		return uuid.Nil, uuid.Nil, errors.New("invalid partition index")
	}

	if partition == 0 {
		return r.start, r.partitions[0], nil
	}

	return r.partitions[partition-1], r.partitions[partition], nil
}
