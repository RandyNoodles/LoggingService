package ratelimiter

import (
	"time"
)

type RateLimiter struct {
	timestampBuffer []uint32 //Ring buffer
	bufferSize      uint32
	writePos        uint32
}

func New(size uint32) *RateLimiter {

	//Create buffer & initialize every value to UINT32_MAX
	buf := make([]uint32, size)
	maxUint32 := ^uint32(0)
	for i := range buf {
		buf[i] = maxUint32
	}

	return &RateLimiter{
		timestampBuffer: buf,
		bufferSize:      size,
		writePos:        0,
	}
}

func (mrb *RateLimiter) IsRateExceeded() bool {
	//Get current Unix time in seconds
	currentSeconds := uint32(time.Now().Unix())

	//Have you sent more than [mrb.bufferSize] messages in the last 60 sec?
	timeElapsed := currentSeconds - mrb.timestampBuffer[mrb.writePos]
	if timeElapsed < 60 {
		return true //If yes, your rate has been exceeded.
	}

	//Else, you're good.
	mrb.timestampBuffer[mrb.writePos] = currentSeconds

	//Increment write position in the ring
	mrb.writePos++
	if mrb.writePos > (mrb.bufferSize - 1) {
		mrb.writePos = 0
	}

	return false
}
