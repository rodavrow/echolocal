package firewall

import (
	"net/netip"
	"slices"
	"testing"
)

func accepted(t *testing.T, local []string, address string) bool {
	t.Helper()
	addr := netip.MustParseAddr(address)
	for _, rule := range egressRules(local) {
		at := slices.Index(rule, "-d")
		if at < 0 || rule[len(rule)-1] != "ACCEPT" {
			continue
		}
		if netip.MustParsePrefix(rule[at+1]).Contains(addr) {
			return true
		}
	}
	return false
}

func TestEgressRulesLetTheLocalNetworkThrough(t *testing.T) {
	for _, address := range []string{
		"192.168.80.1",    // the gateway and its DNS
		"192.168.20.10",   // another VLAN
		"10.1.2.3",        // private
		"172.31.255.254",  // the top of 172.16/12
		"169.254.10.10",   // link-local
		"224.0.0.251",     // mDNS
		"239.255.255.250", // SSDP
		"255.255.255.255", // DHCP broadcast
	} {
		if !accepted(t, local4, address) {
			t.Errorf("%s is refused: it is on the local network", address)
		}
	}
	for _, address := range []string{"fe80::1", "fd12:3456::1", "ff02::fb"} {
		if !accepted(t, local6, address) {
			t.Errorf("%s is refused: it is on the local network", address)
		}
	}
}

func TestEgressRulesRefuseTheInternet(t *testing.T) {
	for _, address := range []string{
		"142.250.27.188", // Google's push service, what a Show kept reaching
		"8.8.8.8",
		"172.32.0.1", // just past 172.16/12
		"100.64.0.1", // carrier-grade NAT is not ours
		"192.169.0.1",
	} {
		if accepted(t, local4, address) {
			t.Errorf("%s is let through: it is not on the local network", address)
		}
	}
	if accepted(t, local6, "2a00:1450:4013:c00::bc") {
		t.Error("a global IPv6 address is let through")
	}
}

func TestEgressRulesEndByRefusingEverythingElse(t *testing.T) {
	rules := egressRules(local4)

	last := rules[len(rules)-1]
	if !slices.Equal(last, []string{Egress, "-j", "REJECT"}) {
		t.Errorf("the last rule is %v, want one refusing whatever is left", last)
	}
	reset := rules[len(rules)-2]
	if !slices.Contains(reset, "tcp-reset") {
		t.Errorf("TCP is refused by %v, want a reset so connections fail at once", reset)
	}
}

func TestEgressRulesLeaveLoopbackAlone(t *testing.T) {
	first := egressRules(local4)[0]

	if !slices.Equal(first, []string{Egress, "-o", "lo", "-j", "ACCEPT"}) {
		t.Errorf("the first rule is %v, want loopback accepted before anything is refused", first)
	}
}
