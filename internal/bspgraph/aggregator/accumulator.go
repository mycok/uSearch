package aggregator

import (
	"math"
	"sync/atomic"
	"unsafe"
)

// Float64Accumulator implements a concurrent-safe accumulator for float64 values.
type Float64Accumulator struct {
	prevSum float64
	currSum float64
}

// Type implements bspgraph.Aggregator.
func (a *Float64Accumulator) Type() string { return "Float64Accumulator" }

// Get returns the current value of the accumulator.
func (a *Float64Accumulator) Get() interface{} {
	return loadFloat64(&a.currSum)
}

// Set the current value of the accumulator.
func (a *Float64Accumulator) Set(val interface{}) {
	for v64 := val.(float64); ; {
		oldCurr := loadFloat64(&a.currSum)
		oldPrevSum := loadFloat64(&a.prevSum)
		swappedCurr := atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.currSum)),
			math.Float64bits(oldCurr),
			math.Float64bits(v64),
		)
		swappedPrev := atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.prevSum)),
			math.Float64bits(oldPrevSum),
			math.Float64bits(v64),
		)

		if swappedCurr && swappedPrev {
			return
		}

	}
}

// Aggregate adds a float64 value to the accumulator.
func (a *Float64Accumulator) Aggregate(val interface{}) {
	for v64 := val.(float64); ; {
		oldV := loadFloat64(&a.currSum)
		newV := oldV + v64

		// Try to update the accumulator's currSum value by copying newV value into the a.currSum
		// field and if successful return.
		if atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.currSum)),
			math.Float64bits(oldV),
			math.Float64bits(newV),
		) {
			return
		}
	}
}

// Delta returns the delta change in the accumulator value since the last time
// it was invoked or the last time that Set was invoked.
func (a *Float64Accumulator) Delta() interface{} {
	for {
		currSum := loadFloat64(&a.currSum)
		prevSum := loadFloat64(&a.prevSum)

		// Try to update the accumulator's prevSum value by copying currSum value into the prevSum
		// field and if successful return the difference.
		if atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.prevSum)),
			math.Float64bits(prevSum),
			math.Float64bits(currSum),
		) {
			return currSum - prevSum
		}
	}
}

// loadFloat64 loads, manipulates and returns a raw float64 value.
func loadFloat64(val *float64) float64 {
	return math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(val))))
}

// IntAccumulator implements a concurrent-safe accumulator for int values.
type IntAccumulator struct {
	prevSum int64
	currSum int64
}

// Type implements bspgraph.Aggregator.
func (a *IntAccumulator) Type() string { return "IntAccumulator" }

// Get returns the current value of the accumulator.
func (a *IntAccumulator) Get() interface{} {
	return int(atomic.LoadInt64(&a.currSum))
}

// Set the current value of the accumulator.
func (a *IntAccumulator) Set(val interface{}) {
	for v64 := int64(val.(int)); ; {
		oldCurr := a.currSum
		oldPrevSum := a.prevSum
		swappedCurr := atomic.CompareAndSwapInt64(&a.currSum, oldCurr, v64)
		swappedPrev := atomic.CompareAndSwapInt64(&a.prevSum, oldPrevSum, v64)

		if swappedCurr && swappedPrev {
			return
		}
	}
}

// Aggregate adds an int value to the accumulator.
func (a *IntAccumulator) Aggregate(val interface{}) {
	_ = atomic.AddInt64(&a.currSum, int64(val.(int)))
}

// Delta returns the delta change in the accumulator value since the last time
// it was invoked or the last time that Set was invoked.
func (a *IntAccumulator) Delta() interface{} {
	for {
		currSum := atomic.LoadInt64(&a.currSum)
		prevSum := atomic.LoadInt64(&a.prevSum)

		if atomic.CompareAndSwapInt64(&a.prevSum, prevSum, currSum) {
			return int(currSum - prevSum)
		}
	}
}
