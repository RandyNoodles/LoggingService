/*
* FILE : 			abuse_prevention.go
* PROJECT : 		SENG2040 - Assignment #3
* PROGRAMMER : 		Woongbeen Lee, Joshua Rice
* FIRST VERSION : 	2025-02-22
* DESCRIPTION :
			AbusePreventionTracker is a class that tracks the number of times
		clients have exceeded their rate limit or sent bad messages.

		All messages return nil if no action has been taken, or an error detailing that
		the IP has been either:
		- Blacklisted due to repeat offenses
		- Is already blacklisted, and how much longer

		Functions provided:
		- CheckIpBlacklist()
		- CheckIpRateLimiter()
		- IncrementBadFormatCounter()
*/

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

// Returns an error stating if IP is either:
// - Blacklisted for N more seconds
// - Newly blacklisted due to repeat offences
// Returns nil if no issue.
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

			//If Blacklist is permanent
			if apt.isBlacklistPermanent {
				return fmt.Errorf("IP address %s has exceeded its message rate limit too many times and has been blacklisted", ipAddress)
			}

			return fmt.Errorf("IP address %s has exceeded its message rate limit too many times. IP address is now banned for %d seconds", ipAddress, apt.blacklistDurationSeconds)
		}
		return fmt.Errorf("IP address %s has exceeded its message rate limit", ipAddress)
	}
	return nil
}

// Returns an error stating if IP blacklisted, and for how much longer
// Returns nil if:
// -IP is no longer on the blacklist
// -IP has served their blacklist duration.
func (apt *AbusePreventionTracker) CheckIPBlacklist(ipAddress string) error {

	if timestamp, exists := apt.blacklistedIPs[ipAddress]; exists {

		//If blacklist is permanent
		if apt.isBlacklistPermanent {
			return fmt.Errorf("IP has been blacklisted")
		}

		//Else, check if it's time to unban them
		durationBanned := uint32(time.Now().Unix()) - timestamp
		if durationBanned >= apt.blacklistDurationSeconds {
			// They've served their time; unban them.
			delete(apt.blacklistedIPs, ipAddress)
		} else {
			// If still banned; calculate the remaining time.
			remaining := apt.blacklistDurationSeconds - durationBanned
			return fmt.Errorf("ip is blacklisted for %v more seconds", remaining)
		}
	}
	return nil
}

// Returns an error if an IP has submitted too many bad messages and has been banned
// Otherwise, increments counter and returns nil
func (apt *AbusePreventionTracker) IncrementBadFormatCount(sourceIp string) error {

	apt.ipBadFormatCount[sourceIp]++
	if apt.ipBadFormatCount[sourceIp] >= apt.badMessageThreshold {
		//Blacklist IP
		apt.blacklistedIPs[sourceIp] = uint32(time.Now().Unix())

		//Reset bad format and rate limiter offence counts
		apt.ipRateLimiters[sourceIp].ResetClientOffenses()
		apt.ipBadFormatCount[sourceIp] = 0

		//If blacklist is permanent
		if apt.isBlacklistPermanent {
			return fmt.Errorf("IP has exceeded it's malformed message threshold and been blacklisted")
		}

		//If IP will be un-blacklisted in the future
		return fmt.Errorf("source has exceeded it's bad message threshold and will be blacklisted for %d seconds", apt.blacklistDurationSeconds)
	}

	return nil
}
