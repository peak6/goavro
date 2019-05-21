package goavro

import (
	"fmt"
	"math"
	"math/big"
	"time"
)

func StringNativeFromBinary(buf []byte) (string, []byte, error) {
	thing, newBuf, err := stringNativeFromBinary(buf)
	if err != nil {
		return "", buf, err
	}
	return thing.(string), newBuf, nil
}

func LongNativeFromBinary(buf []byte) (int64, []byte, error) {
	thing, newBuf, err := longNativeFromBinary(buf)
	if err != nil {
		return 0, buf, err
	}
	return thing.(int64), newBuf, nil
}

func IntNativeFromBinary(buf []byte) (int32, []byte, error) {
	thing, newBuf, err := intNativeFromBinary(buf)
	if err != nil {
		return 0, buf, err
	}
	return thing.(int32), newBuf, nil
}

func DoubleNativeFromBinary(buf []byte) (float64, []byte, error) {
	thing, newBuf, err := doubleNativeFromBinary(buf)
	if err != nil {
		return 0, buf, err
	}
	return thing.(float64), newBuf, nil
}

func FloatNativeFromBinary(buf []byte) (float32, []byte, error) {
	thing, newBuf, err := floatNativeFromBinary(buf)
	if err != nil {
		return 0, buf, err
	}
	return thing.(float32), newBuf, nil
}

func BoolNativeFromBinary(buf []byte) (bool, []byte, error) {
	thing, newBuf, err := booleanNativeFromBinary(buf)
	if err != nil {
		return false, buf, err
	}
	return thing.(bool), newBuf, nil
}

func NativeFromBinaryDecimalBytes(buf []byte, precision int, scale int) (*big.Rat, []byte, error) {
	thing, newBuf, err := nativeFromDecimalBytes(bytesNativeFromBinary, precision, scale)(buf)
	if err != nil {
		return &big.Rat{}, buf, err
	}
	return thing.(*big.Rat), newBuf, nil

}

func NativeFromBinaryDate(buf []byte) (time.Time, []byte, error) {
	thing, newBuf, err := nativeFromDate(intNativeFromBinary)(buf)
	if err != nil {
		return time.Time{}, buf, err
	}
	return thing.(time.Time), newBuf, nil
}

func DecodeBlockCount(buf []byte) (int64, []byte, error) {
	// block count and block size
	var value interface{}
	var err error
	newBuf := buf
	if value, newBuf, err = longNativeFromBinary(newBuf); err != nil {
		return 0, buf, fmt.Errorf("cannot decode binary array block count: %s", err)
	}
	blockCount := value.(int64)
	if blockCount < 0 {
		// NOTE: A negative block count implies there is a long encoded
		// block size following the negative block count. We have no use
		// for the block size in this decoder, so we read and discard
		// the value.
		if blockCount == math.MinInt64 {
			// The minimum number for any signed numerical type can never be made positive
			return 0, buf, fmt.Errorf("cannot decode binary array with block count: %d", blockCount)
		}
		blockCount = -blockCount // convert to its positive equivalent
		if _, newBuf, err = longNativeFromBinary(newBuf); err != nil {
			return 0, buf, fmt.Errorf("cannot decode binary array block size: %s", err)
		}
	}
	// Ensure block count does not exceed some sane value.
	if blockCount > MaxBlockCount {
		return 0, buf, fmt.Errorf("cannot decode binary array when block count exceeds MaxBlockCount: %d > %d", blockCount, MaxBlockCount)
	}
	return blockCount, newBuf, nil
}
