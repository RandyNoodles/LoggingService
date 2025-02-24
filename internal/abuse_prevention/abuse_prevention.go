package abuseprevention

import (
	"LoggingService/config"
	ratelimiter "LoggingService/internal/abuse_prevention/rateLimiter"
	"fmt"
	"time"
)

type AbusePreventionTracker struct {
	ipRateLimiters           map[string]*ratelimiter.RateLimiter
	blacklistedIPs           map[string]uint32
	ipBadFormatCount         map[string]uint32
	ipLimitPerMin            uint32
	blacklistDurationSeconds uint32
	isBlacklistPermanent     bool
	badMessageThreshold      uint32
}

func New(protocolConfig config.ProtocolSettings) *AbusePreventionTracker {

	//Init new maps for rate limiters, blacklistedIps
	newTracker := &AbusePreventionTracker{
		ipRateLimiters:           make(map[string]*ratelimiter.RateLimiter),
		blacklistedIPs:           make(map[string]uint32),
		ipBadFormatCount:         make(map[string]uint32),
		ipLimitPerMin:            uint32(protocolConfig.IpMessagesPerMinute),
		isBlacklistPermanent:     protocolConfig.BlacklistPermanent,
		blacklistDurationSeconds: uint32(protocolConfig.BlacklistDurationSeconds),
		badMessageThreshold:      uint32(protocolConfig.BadMessageBlacklistThreshold),
	}

	//Fill blacklistedIps map IP addresses & the timestamp they were banned.
	for _, ip := range protocolConfig.BlacklistedIPs {
		newTracker.blacklistedIPs[ip] = uint32(time.Now().Unix())
	}

	return newTracker
}

func (apt *AbusePreventionTracker) CheckIPRateLimiter(ipAddress string) error {

	//If IP doesn't exist in our records yet, register them
	_, exists := apt.ipRateLimiters[ipAddress]
	if !exists {
		apt.ipRateLimiters[ipAddress] = ratelimiter.New(apt.ipLimitPerMin)
	}

	//Check if they've exceeded their messages per min limit
	rejected, clientOffenses := apt.ipRateLimiters[ipAddress].IsRateExceeded()
	if rejected {
		//If they've exceeded the allowed threshold, ban them.
		if clientOffenses >= apt.badMessageThreshold {
			apt.blacklistedIPs[ipAddress] = uint32(time.Now().Unix())

			//Reset bad format and rate limiter offence counts
			apt.ipRateLimiters[ipAddress].ResetClientOffenses()
			apt.ipBadFormatCount[ipAddress] = 0

			return fmt.Errorf("IP address %s has exceeded its message rate limit too many times. IP address is now banned for %d seconds", ipAddress, apt.blacklistDurationSeconds)
		}
		return fmt.Errorf("IP address %s has exceeded its message rate limit", ipAddress)
	}
	return nil
}

func (apt *AbusePreventionTracker) CheckIPBlacklist(ipAddress string) error {

	if timestamp, exists := apt.blacklistedIPs[ipAddress]; exists {
		durationBanned := uint32(time.Now().Unix()) - timestamp
		if durationBanned >= apt.blacklistDurationSeconds {
			// They've served their time; unban them.
			delete(apt.blacklistedIPs, ipAddress)
		} else {
			// Still banned; calculate the remaining time.
			remaining := apt.blacklistDurationSeconds - durationBanned
			return fmt.Errorf("ip is blacklisted for %v more seconds", remaining)
		}
	}
	return nil
}

func (apt *AbusePreventionTracker) IncrementBadFormatCount(sourceIp string) error {

	apt.ipBadFormatCount[sourceIp]++
	if apt.ipBadFormatCount[sourceIp] >= apt.badMessageThreshold {
		//Blacklist IP
		apt.blacklistedIPs[sourceIp] = uint32(time.Now().Unix())

		//Reset bad format and rate limiter offence counts
		apt.ipRateLimiters[sourceIp].ResetClientOffenses()
		apt.ipBadFormatCount[sourceIp] = 0
		return fmt.Errorf("source has exceeded it's bad message threshold and will be blacklisted for %d seconds", apt.blacklistDurationSeconds)
	}

	return nil
}
