package main

import (
	"github.com/miekg/dns"
	"github.com/patrickmn/go-cache"
	"time"
)

var dnsCache = cache.New(cache.DefaultExpiration, 10*time.Minute)

func getMinTTL(answer []dns.RR) uint32 {
	var minTTL uint32
	for i, a := range answer {
		ttl := a.Header().Ttl
		if i == 0 {
			minTTL = ttl
			continue
		}
		if ttl < minTTL {
			minTTL = ttl
		}
	}
	return minTTL
}

// SetCache set a dns question/msg cache to the dohproxy in-memory cache
func SetCache(question string, msg *dns.Msg) {
	if len(msg.Answer) == 0 {
		return
	}

	dnsCache.Set(question, msg, time.Duration(getMinTTL(msg.Answer))*time.Second)
}

// GetCache get a dns cache by question string
func GetCache(question string, id uint16) (*dns.Msg, bool) {
	if msg, expiration, found := dnsCache.GetWithExpiration(question); found {
		newMsg := msg.(*dns.Msg)

		// set new ttl
		minTTL := getMinTTL(newMsg.Answer)
		ttlOffset := minTTL - uint32(time.Until(expiration).Seconds())
		for _, answer := range newMsg.Answer {
			answer.Header().Ttl -= ttlOffset
		}

		// set new id
		newMsg.Id = id

		return newMsg, true
	}
	return nil, false
}
