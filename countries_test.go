package sks_spider

import (
	"net"
	"testing"
)

const checkSksHostname = "sks-peer.spodhuis.org"
const checkSksIPCount = 2
const checkSksCountry = "NL"
const checkSksExpectIPv6HasCountry = false

func TestCountrySpodhuis(t *testing.T) {
	ipList, err := net.LookupHost(checkSksHostname)
	if err != nil {
		t.Fatalf("LookupHost(%s) failed: %s", checkSksHostname, err)
	}
	if len(ipList) != checkSksIPCount {
		t.Fatalf("Wrong number of IP addresses for \"%s\": expected %d got %d",
			checkSksHostname, checkSksIPCount, len(ipList))
	}
	var expectSucceed bool
	for _, ip := range ipList {
		switch {
		case net.ParseIP(ip).To4() != nil:
			expectSucceed = true
		default:
			expectSucceed = checkSksExpectIPv6HasCountry
		}
		country, err := CountryForIPString(ip)
		if err != nil {
			if expectSucceed {
				t.Fatalf("Failed to resolve country for [%s] (from \"%s\"): %s",
					ip, checkSksHostname, err)
			}
			continue
		}
		if !expectSucceed {
			t.Fatalf("Unexpectedly resolved country for [%s] (from \"%s\")",
				ip, checkSksHostname)
		}
		if country != checkSksCountry {
			t.Fatalf("Host \"%s\" IP [%s]: expected country \"%s\", got \"%s\"",
				checkSksHostname, ip, checkSksCountry, country)
		}
	}
}

func TestCountrySets(t *testing.T) {
	set := NewCountrySet("us,nl,uk")
	for _, country := range []string{"us", "nl", "uk", "NL", "Us", "uK"} {
		if !set.HasCountry(country) {
			t.Fatalf("Countryset missing country \"%s\"", country)
		}
	}
	for _, country := range []string{"au", "", " ", "GB"} {
		if set.HasCountry(country) {
			t.Fatalf("Countryset unexpectedly has country \"%s\"", country)
		}
	}
	if set.String() != "NL,UK,US" {
		t.Fatalf("Countryset stringification unsorted: %s", set)
	}
	t.Logf("Countryset OK: %s", set)
}
