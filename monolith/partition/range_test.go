package partition

import (
	"github.com/google/uuid"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(new(RangeTestSuite))

type RangeTestSuite struct{}

func (s *RangeTestSuite) TestRangeErrors(c *check.C) {
	_, err := NewRange(
		1,
		uuid.MustParse("40000000-0000-0000-0000-000000000000"),
		uuid.MustParse("00000000-0000-0000-0000-000000000000"),
	)
	c.Assert(err, check.ErrorMatches,
		"range start UUID must be less than the end UUID",
	)

	_, err = NewRange(
		0,
		uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		uuid.MustParse("40000000-0000-0000-0000-000000000000"),
	)
	c.Assert(err, check.ErrorMatches,
		"number of partitions must be at least equal to 1",
	)
}

func (s *RangeTestSuite) TestEvenSplit(c *check.C) {
	r, err := NewFullRange(4)
	c.Assert(err, check.IsNil)

	expectedRangePartitions := [][2]uuid.UUID{
		{
			uuid.MustParse("00000000-0000-0000-0000-000000000000"),
			uuid.MustParse("40000000-0000-0000-0000-000000000000"),
		},
		{
			uuid.MustParse("40000000-0000-0000-0000-000000000000"),
			uuid.MustParse("80000000-0000-0000-0000-000000000000"),
		},
		{
			uuid.MustParse("80000000-0000-0000-0000-000000000000"),
			uuid.MustParse("c0000000-0000-0000-0000-000000000000"),
		},
		{
			uuid.MustParse("c0000000-0000-0000-0000-000000000000"),
			uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		},
	}

	for i, partition := range expectedRangePartitions {
		c.Logf("range: %d", i)
		from, to, err := r.PartitionRange(i)
		c.Assert(err, check.IsNil)
		c.Check(from.String(), check.Equals, partition[0].String())
		c.Check(to.String(), check.Equals, partition[1].String())
	}
}

func (s *RangeTestSuite) TestOddSplit(c *check.C) {
	r, err := NewFullRange(3)
	c.Assert(err, check.IsNil)

	expectedRangePartitions := [][2]uuid.UUID{
		{
			uuid.MustParse("00000000-0000-0000-0000-000000000000"),
			uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		},
		{
			uuid.MustParse("55555555-5555-5555-5555-555555555555"),
			uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		},
		{
			uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		},
	}

	for i, partition := range expectedRangePartitions {
		c.Logf("range: %d", i)
		from, to, err := r.PartitionRange(i)
		c.Assert(err, check.IsNil)
		c.Check(from.String(), check.Equals, partition[0].String())
		c.Check(to.String(), check.Equals, partition[1].String())
	}
}

func (s *RangeTestSuite) TestPartitionExtentsError(c *check.C) {
	r, err := NewRange(
		1,
		uuid.MustParse("11111111-0000-0000-0000-000000000000"),
		uuid.MustParse("55555555-0000-0000-0000-000000000000"),
	)
	c.Assert(err, check.IsNil)

	_, _, err = r.PartitionRange(1)
	c.Assert(err, check.ErrorMatches, "invalid partition index")
}
