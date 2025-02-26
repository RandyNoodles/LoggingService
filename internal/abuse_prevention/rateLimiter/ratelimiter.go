/*
* FILE : 			ratelimiter.go
* FIRST VERSION : 	2025-02-22
* DESCRIPTION :
			RateLimiter is basically a simplified ring buffer that tracks
		how many messages have been received by a source in the last minute.

		- Ring buffer is allocated to size "num_messages_per_ip_per_min" from config.json

		When IsRateExceeded() is used:
		- Current timestamp is compared to the next available spot in the ring
			- If time difference < 60s, message is rejected and the offense counter is incremented
			- If >= 60s, timestamp is recorded and message is allowed

		All unix timestamps are truncated from uint64 --> uint32.
		- Saves memory at scale
		- Still allows 80 more years of UNIX time
*/

package ratelimiter

import (
	"time"
)

type RateLimiter struct {
	timestampBuffer []uint32 //Ring buffer
	bufferSize      uint32
	writePos        uint32
	clientOffenses  uint32
}

func New(msgPerMin uint32) *RateLimiter {

	//Create buffer & initialize every value to UINT32_MAX
	buf := make([]uint32, msgPerMin)
	maxUint32 := ^uint32(0)
	for i := range buf {
		buf[i] = maxUint32
	}

	return &RateLimiter{
		timestampBuffer: buf,
		bufferSize:      msgPerMin,
		writePos:        0,
		clientOffenses:  0,
	}
}

func (mrb *RateLimiter) IsRateExceeded() (bool, uint32) {
	//Get current Unix time in seconds
	currentSeconds := uint32(time.Now().Unix())

	//Have you sent more than [mrb.bufferSize] messages in the last 60 sec?
	timeElapsed := currentSeconds - mrb.timestampBuffer[mrb.writePos]
	if timeElapsed < 60 {

		mrb.clientOffenses++
		return true, mrb.clientOffenses //If yes, your rate has been exceeded.
	}

	//Else, you're good.
	mrb.timestampBuffer[mrb.writePos] = currentSeconds

	//Increment write position in the ring
	mrb.writePos++
	if mrb.writePos > (mrb.bufferSize - 1) {
		mrb.writePos = 0
	}

	return false, mrb.clientOffenses
}

func (mrb *RateLimiter) IncrementClientOffenses() uint32 {
	mrb.clientOffenses++
	return mrb.clientOffenses
}

func (mrb *RateLimiter) ResetClientOffenses() {
	mrb.clientOffenses = 0
	for i := range mrb.timestampBuffer {
		mrb.timestampBuffer[i] = ^uint32(0)
	}
}
