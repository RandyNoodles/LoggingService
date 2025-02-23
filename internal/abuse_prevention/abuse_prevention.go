package abuseprevention

import (
	"LoggingService/config"
	ratelimiter "LoggingService/internal/abuse_prevention/rateLimiter"
	"fmt"
	"sync"
	"time"
)

var abusePreventionMutex sync.Mutex

type AbusePreventionTracker struct {
	//Rate limiters are ring buffers that can ensure a source doesn't send more than N messages per min
	ipRateLimiters     map[string]*ratelimiter.RateLimiter
	sourceRateLimiters map[string]*ratelimiter.RateLimiter
	//Blacklisted IPs/IDs, along with the timestamp of when they were banned (so we can potentially unban later)
	blacklistedIPs       map[string]uint32
	blacklistedSourceIds map[string]uint32
	//Tracks the number of bad formatted messages
	sourceBadFormatCount map[string]uint32
	//Rates loaded via config.json
	ipLimitPerMin            uint32
	sourceIdLimitPerMin      uint32
	blacklistDurationSeconds uint32
	isBlacklistPermanent     bool
	badMessageThreshold      uint32
}

func New(protocolConfig config.ProtocolSettings) *AbusePreventionTracker {

	//Init new maps for rate limiters, blacklistedIps
	newTracker := &AbusePreventionTracker{
		ipRateLimiters:           make(map[string]*ratelimiter.RateLimiter),
		sourceRateLimiters:       make(map[string]*ratelimiter.RateLimiter),
		blacklistedIPs:           make(map[string]uint32),
		blacklistedSourceIds:     make(map[string]uint32),
		sourceBadFormatCount:     make(map[string]uint32),
		ipLimitPerMin:            uint32(protocolConfig.IpMessagesPerMinute),
		sourceIdLimitPerMin:      uint32(protocolConfig.SourceMessagesPerMinute),
		isBlacklistPermanent:     protocolConfig.BlacklistPermanent,
		blacklistDurationSeconds: uint32(protocolConfig.BlacklistDurationSeconds),
		badMessageThreshold:      uint32(protocolConfig.BadMessageBlacklistThreshold),
	}

	//Fill blacklistedIps map IP addresses & the timestamp they were banned.
	for _, ip := range protocolConfig.BlacklistedIPs {
		newTracker.blacklistedIPs[ip] = uint32(time.Now().Unix())
	}
	for _, id := range protocolConfig.BlacklistedSourceIds {
		newTracker.blacklistedSourceIds[id] = uint32(time.Now().Unix())
	}

	return newTracker
}

func (apt *AbusePreventionTracker) CheckSourceRateLimiter(sourceId, ipAddress string) error {
	abusePreventionMutex.Lock()
	defer abusePreventionMutex.Unlock()

	rejected, clientOffenses := apt.sourceRateLimiters[sourceId].IsRateExceeded()
	if rejected {
		if clientOffenses >= apt.badMessageThreshold {
			// Ban the IP if the client has exceeded the bad message threshold.
			apt.blacklistedIPs[ipAddress] = uint32(time.Now().Unix())
			return fmt.Errorf("IP address %s has exceeded its message rate limit too many times. IP address is now banned for %d seconds", ipAddress, apt.blacklistDurationSeconds)
		}
		return fmt.Errorf("source_id '%s' has exceeded its message rate limit", sourceId)
	}
	return nil
}

func (apt *AbusePreventionTracker) CheckIPRateLimiter(ipAddress string) error {
	abusePreventionMutex.Lock()
	defer abusePreventionMutex.Unlock()

	rejected, clientOffenses := apt.ipRateLimiters[ipAddress].IsRateExceeded()
	if rejected {
		if clientOffenses >= apt.badMessageThreshold {
			// Ban the IP if the client has exceeded the bad message threshold.
			apt.blacklistedIPs[ipAddress] = uint32(time.Now().Unix())
			return fmt.Errorf("IP address %s has exceeded its message rate limit too many times. IP address is now banned for %d seconds", ipAddress, apt.blacklistDurationSeconds)
		}
		return fmt.Errorf("IP address %s has exceeded its message rate limit", ipAddress)
	}
	return nil
}

func (apt *AbusePreventionTracker) RegisterSource(ipAddress string, sourceId string) {
	abusePreventionMutex.Lock()
	defer abusePreventionMutex.Unlock()

	//Does the client already exist?
	//If not, register.

	_, exists := apt.sourceRateLimiters[sourceId]
	if !exists {
		apt.sourceRateLimiters[sourceId] = ratelimiter.New(apt.sourceIdLimitPerMin)
	}

	_, exists = apt.ipRateLimiters[ipAddress]
	if !exists {
		apt.ipRateLimiters[ipAddress] = ratelimiter.New(apt.ipLimitPerMin)
	}

}

func (apt *AbusePreventionTracker) CheckIPBlacklist(ipAddress string) error {
	abusePreventionMutex.Lock()
	defer abusePreventionMutex.Unlock()

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

func (apt *AbusePreventionTracker) CheckSourceIDBlacklist(sourceId string) error {
	abusePreventionMutex.Lock()
	defer abusePreventionMutex.Unlock()

	if timestamp, exists := apt.blacklistedSourceIds[sourceId]; exists {
		durationBanned := uint32(time.Now().Unix()) - timestamp
		if durationBanned >= apt.blacklistDurationSeconds {
			// They've served their time; unban them.
			delete(apt.blacklistedSourceIds, sourceId)
		} else {
			// Still banned; calculate the remaining time.
			remaining := apt.blacklistDurationSeconds - durationBanned
			return fmt.Errorf("source_id is blacklisted for %v more seconds", remaining)
		}
	}
	return nil
}

func (apt *AbusePreventionTracker) IncrementBadFormatCount(sourceIp string) error {
	abusePreventionMutex.Lock()
	defer abusePreventionMutex.Unlock()

	apt.sourceBadFormatCount[sourceIp]++
	if apt.sourceBadFormatCount[sourceIp] >= apt.badMessageThreshold {
		apt.blacklistedIPs[sourceIp] = uint32(time.Now().Unix())
		return fmt.Errorf("source has exceeded it's bad message threshold and will be blacklisted for %d seconds", apt.blacklistDurationSeconds)
	}

	return nil
}
