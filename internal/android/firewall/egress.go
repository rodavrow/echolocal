package firewall

import "errors"

const ip6tables = "/system/bin/ip6tables"

// Egress holds what the device may send while it is kept to the local network. Empty, it lets
// everything through.
const Egress = "ECHOLOCAL_EGRESS"

// The local network is every private, link-local and multicast range rather than the device's own
// subnet: Home Assistant and the things a display talks to often sit on another VLAN.
var (
	local4 = []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"224.0.0.0/4",
		"255.255.255.255/32",
	}
	local6 = []string{"fe80::/10", "fc00::/7", "ff00::/8"}
)

var localOnly bool

// LocalOnly refuses everything the device sends beyond the local network while on, whichever
// process sends it, and lifts that when off. Refused connections fail at once instead of timing
// out.
func LocalOnly(on bool) error {
	ready.Lock()
	defer ready.Unlock()
	localOnly = on
	return keepLocal(on)
}

func keepLocal(on bool) error {
	return errors.Join(egress(iptables, local4, on), egress(ip6tables, local6, on))
}

func egress(tables string, local []string, on bool) error {
	// -N fails when the chain already exists.
	if err := run(tables, "-N", Egress); err != nil && !errors.Is(err, errExists) {
		return err
	}
	if err := run(tables, "-F", Egress); err != nil {
		return err
	}
	if on {
		for _, rule := range egressRules(local) {
			if err := run(tables, append([]string{"-A"}, rule...)...); err != nil {
				return err
			}
		}
	}
	jump := []string{"OUTPUT", "-j", Egress}
	there, err := hasIn(tables, jump)
	if err != nil || there {
		return err
	}
	return run(tables, append([]string{"-I"}, jump...)...)
}

// egressRules accept loopback and the local ranges, then refuse the rest: TCP with a reset, so a
// connection fails at once, anything else as unreachable.
func egressRules(local []string) [][]string {
	rules := [][]string{{Egress, "-o", "lo", "-j", "ACCEPT"}}
	for _, network := range local {
		rules = append(rules, []string{Egress, "-d", network, "-j", "ACCEPT"})
	}
	return append(rules,
		[]string{Egress, "-p", "tcp", "-j", "REJECT", "--reject-with", "tcp-reset"},
		[]string{Egress, "-j", "REJECT"},
	)
}
