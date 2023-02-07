package aggregator

import (
	"math"
	"sync/atomic"
	"unsafe"
)

// Float64Accumulator is a concurrent-safe accumulator for float64 values.
// It satisfies the bsp.Aggregator interface.
type Float64Accumulator struct {
	prevSum float64
	currSum float64
}

// Type returns the type of this accumulator as a string.
func (a *Float64Accumulator) Type() string {
	return "Float64Accumulator"
}

// Get retrieves the current accumulator value.
func (a *Float64Accumulator) Get() interface{} {
	return loadFloat64(&a.currSum)
}

// Set the accumulator's fields to the specified value.
func (a *Float64Accumulator) Set(val interface{}) {
	for value := val.(float64); ; {
		oldCurrSum := loadFloat64(&a.currSum)
		oldPrevSum := loadFloat64(&a.prevSum)

		swappedCurrSum := atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.currSum)),
			math.Float64bits(oldCurrSum), math.Float64bits(value),
		)
		swappedPrevSum := atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.prevSum)),
			math.Float64bits(oldPrevSum), math.Float64bits(value),
		)

		if swappedCurrSum && swappedPrevSum {
			return
		}
	}
}

// Aggregate updates the accumulator's current sum to the provided value.
func (a *Float64Accumulator) Aggregate(val interface{}) {
	// Loop indefinitely until a compare-swap operation is successful.
	for value := val.(float64); ; {
		oldValue := loadFloat64(&a.currSum)
		newValue := oldValue + value

		if atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.currSum)),
			math.Float64bits(oldValue), math.Float64bits(newValue),
		) {
			return
		}
	}
}

// Delta returns the change in the accumulator's value since the last
// call to Delta or Set.
func (a *Float64Accumulator) Delta() interface{} {
	for {
		currSum := loadFloat64(&a.currSum)
		prevSum := loadFloat64(&a.prevSum)

		// Try to update the accumulator's prevSum value by copying currSum
		// value into the prevSum field and if successful return the difference.
		if atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.prevSum)),
			math.Float64bits(prevSum), math.Float64bits(currSum),
		) {

			return currSum - prevSum
		}
	}
}

func loadFloat64(val *float64) float64 {
	// Convert the float64 pointer value to a generic pointer type, cast that
	// generic pointer type to an uint64 pointer type and atomically load and
	// convert the uint64 bits into a float64 data type value.
	//
	// Note: This is done because the atomic package doesn't directly support
	// float64 types.
	return math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(val))))
}

// IntAccumulator is a concurrent-safe accumulator for int64 values.
// It satisfies the bsp.Aggregator interface.
type IntAccumulator struct {
	prevSum int64
	currSum int64
}

// Type returns the type of this accumulator as a string.
func (a *IntAccumulator) Type() string {
	return "Float64Accumulator"
}

// Get retrieves the current accumulator value.
func (a *IntAccumulator) Get() interface{} {
	return int(atomic.LoadInt64(&a.currSum))
}

// Set the accumulator's fields to the specified value.
func (a *IntAccumulator) Set(val interface{}) {
	for value := int64(val.(int)); ; {
		oldCurrSum := a.currSum
		oldPrevSum := a.prevSum

		swappedCurrSum := atomic.CompareAndSwapInt64(&a.currSum, oldCurrSum, value)
		swappedPrevSum := atomic.CompareAndSwapInt64(&a.prevSum, oldPrevSum, value)

		if swappedCurrSum && swappedPrevSum {
			return
		}
	}
}

// Aggregate updates the accumulator's current sum to the provided value.
func (a *IntAccumulator) Aggregate(val interface{}) {
	_ = atomic.AddInt64(&a.currSum, int64(val.(int)))
}

// Delta returns the change in the accumulator's value since the last
// call to Delta or Set.
func (a *IntAccumulator) Delta() interface{} {
	for {
		currSum := atomic.LoadInt64(&a.currSum)
		prevSum := atomic.LoadInt64(&a.prevSum)

		// Try to update the accumulator's prevSum value by copying currSum
		// value into the prevSum field and if successful return the difference.
		if atomic.CompareAndSwapInt64(&a.prevSum, prevSum, currSum) {
			return int(currSum - prevSum)
		}
	}
}
