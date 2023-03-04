package rpc_test

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	check "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func encodeTimestamp(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}


