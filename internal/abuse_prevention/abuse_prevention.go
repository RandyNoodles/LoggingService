package abuseprevention

import (
	"LoggingService/config"
	ratelimiter "LoggingService/internal/abuse_prevention/rateLimiter"
	"fmt"
	"time"
)

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

func (apt *AbusePreventionTracker) IsValidMessage(ipAddress string, sourceId string) error {

	result := apt.CheckBlacklist(ipAddress, sourceId)
	if result != nil {
		return result
	}

	apt.RegisterSource(ipAddress, sourceId)

	//Check for rate limit exceeded for source
	rejected, clientOffenses := apt.sourceRateLimiters[sourceId].IsRateExceeded()
	if rejected {
		//If the client has exceeded bad message threshold, ban 'em
		if clientOffenses >= apt.badMessageThreshold {

			apt.blacklistedIPs[ipAddress] = uint32(time.Now().Unix())
			return fmt.Errorf("IP address %s has exceeded its message rate limit too many times. IP address is now banned for %d seconds", ipAddress, apt.blacklistDurationSeconds)
		}
		return fmt.Errorf("source_id '%s' has exceeded its message rate limit", sourceId)
	}

	//Then check for rate limit of IP
	rejected, clientOffenses = apt.ipRateLimiters[ipAddress].IsRateExceeded()
	if rejected {
		//If the client has exceeded bad message threshold, ban 'em
		if clientOffenses >= apt.badMessageThreshold {
			clientOffenses = 0
			apt.blacklistedIPs[ipAddress] = uint32(time.Now().Unix())
			return fmt.Errorf("IP address %s has exceeded its message rate limit too many times. IP address is now banned for %d seconds", ipAddress, apt.blacklistDurationSeconds)
		}
		return fmt.Errorf("IP address %s has exceeded its message rate limit", ipAddress)
	}

	return nil
}

func (apt *AbusePreventionTracker) RegisterSource(ipAddress string, sourceId string) {
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

func (apt *AbusePreventionTracker) CheckBlacklist(ipAddress string, sourceId string) error {
	//Are they in the list of blacklisted ips?
	if timestamp, exists := apt.blacklistedIPs[ipAddress]; exists {

		//How long they've been banned for
		durationBanned := (uint32(time.Now().Unix()) - timestamp)

		//If they've served their time, unban them
		if durationBanned >= apt.blacklistDurationSeconds {
			delete(apt.blacklistedIPs, ipAddress)
		} else {
			//Else reject
			return fmt.Errorf("ip is blacklisted for %v more seconds", durationBanned-apt.blacklistDurationSeconds)
		}
	}

	if timestamp, exists := apt.blacklistedSourceIds[sourceId]; exists {
		//How long they've been banned for
		durationBanned := (uint32(time.Now().Unix()) - timestamp)

		//If they've served their time, unban them
		if durationBanned >= apt.blacklistDurationSeconds {
			delete(apt.blacklistedSourceIds, sourceId)
		} else {
			//Else reject
			return fmt.Errorf("source_id is blacklisted for %v more seconds", durationBanned-apt.blacklistDurationSeconds)
		}
	}

	return nil
}

func (apt *AbusePreventionTracker) IncrementBadFormatCount(sourceId string) error {

	apt.sourceBadFormatCount[sourceId]++
	if apt.sourceBadFormatCount[sourceId] >= apt.badMessageThreshold {
		apt.blacklistedSourceIds[sourceId] = uint32(time.Now().Unix())
		return fmt.Errorf("source has exceeded it's bad message threshold and will be blacklisted for %d seconds", apt.blacklistDurationSeconds)
	}

	return nil
}
