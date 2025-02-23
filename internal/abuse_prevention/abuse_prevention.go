package abuseprevention

import (
	"LoggingService/config"
	"LoggingService/internal/abuse_prevention/ratelimiter"
	"fmt"
	"time"
)

type AbusePreventionTracker struct {
	//Rate limiters are ring buffers that can ensure a source doesn't send more than N messages per min
	ipRateLimiters     map[string]*ratelimiter.RateLimiter
	sourceRateLimiters map[string]*ratelimiter.RateLimiter
	//Blacklisted IPs, along with the timestamp of when they were banned (so we can potentially unban later)
	blacklistedIPs map[string]uint32
	//Rates loaded via config.json
	ipLimitPerMin            uint32
	sourceIdLimitPerMin      uint32
	blacklistDurationSeconds uint32
	isBlacklistPermanent     bool
}

func New(protocolConfig config.ProtocolSettings) *AbusePreventionTracker {

	//Init new maps for rate limiters, blacklistedIps
	newTracker := &AbusePreventionTracker{
		ipRateLimiters:           make(map[string]*ratelimiter.RateLimiter),
		sourceRateLimiters:       make(map[string]*ratelimiter.RateLimiter),
		blacklistedIPs:           make(map[string]uint32),
		ipLimitPerMin:            uint32(protocolConfig.IpMessagesPerMinute),
		sourceIdLimitPerMin:      uint32(protocolConfig.SourceMessagesPerMinute),
		isBlacklistPermanent:     protocolConfig.BlacklistPermanent,
		blacklistDurationSeconds: uint32(protocolConfig.BlacklistDurationSeconds),
	}

	//Fill blacklistedIps map IP addresses & the timestamp they were banned.
	for _, ip := range protocolConfig.BlacklistedIPs {
		newTracker.blacklistedIPs[ip] = uint32(time.Now().Unix())
	}

	return newTracker
}

func (apt *AbusePreventionTracker) IsValidMessage(ipAddress string, source_id string) error {

	//Are they in the list of blacklisted ips?
	if timestamp, exists := apt.blacklistedIPs[ipAddress]; exists {

		//How long they've been banned for
		durationBanned := (uint32(time.Now().Unix()) - timestamp)

		//If they've served their time, unban them
		if durationBanned >= apt.blacklistDurationSeconds {
			delete(apt.blacklistedIPs, ipAddress)
		} else {
			//Else reject
			return fmt.Errorf("Ip is blacklisted for %v more seconds", durationBanned-apt.blacklistDurationSeconds)
		}
	}

	//Check for rate limit exceeded for source
	rejected := apt.sourceRateLimiters[source_id].IsRateExceeded()
	if rejected {
		return fmt.Errorf("source_id '%s' has exceeded its message rate limit", source_id)
	}

	//Then check for rate limit of IP
	rejected = apt.ipRateLimiters[ipAddress].IsRateExceeded()
	if rejected {
		return fmt.Errorf("IP address %s has exceeded its message rate limit", ipAddress)
	}

	return nil
}
