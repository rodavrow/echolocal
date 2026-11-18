package network

import (
	"log/slog"
	"sync"

	"github.com/ygelfand/libcountertop/pkg/network/policy"
	sharedwifi "github.com/ygelfand/libcountertop/pkg/network/wifi"
	sharedcomponent "github.com/ygelfand/libcountertop/pkg/runtime/component"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"

	"github.com/ygelfand/echolocal/internal/android/firewall"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/dhcp"
	"github.com/ygelfand/echolocal/internal/feature/homeassistant"
	hwwifi "github.com/ygelfand/echolocal/internal/hardware/wifi"
)

func init() {
	component.Register(sharedcomponent.Network, Get, sharedcomponent.Order(5))
}

type Network struct {
	*policy.Policy
	wifi *sharedwifi.Mover
}

var get = sync.OnceValue(func() *Network {
	n := &Network{
		Policy: policy.New(config.NetworkSection),
		wifi:   sharedwifi.NewMover(hwwifi.Get(), dhcp.Get().Fresh),
	}
	n.DownloadThrough(homeassistant.Get().Downloads())
	component.Settings.Add(n.Settings())
	// Applied at every start, open included: rules left by an earlier run must not outlive a
	// setting that has since changed.
	local := n.Internet().Local()
	keepLocal(local)
	keep := onChange(local, keepLocal)
	n.Changed.Listen(func(s schema.Network) { keep(s.Internet.Local()) })
	return n
})

// keepLocal has the firewall refuse what the device sends beyond the local network, or lifts that.
func keepLocal(on bool) {
	if err := firewall.LocalOnly(on); err != nil {
		slog.Error("keeping the device to the local network failed", "local_only", on, "err", err)
		return
	}
	slog.Info("internet access", "local_only", on)
}

// onChange returns a func to call with a value each time it may have changed. It calls do only
// when the value differs from the last one, starting from was: the policy announces every network
// change, and each call rebuilds the firewall chain.
func onChange(was bool, do func(bool)) func(now bool) {
	var mu sync.Mutex
	return func(now bool) {
		mu.Lock()
		changed := now != was
		was = now
		mu.Unlock()
		if changed {
			do(now)
		}
	}
}

func Get() *Network                          { return get() }
func (n *Network) Wifi() *sharedwifi.Mover   { return n.wifi }
func (n *Network) Restore(cfg config.Config) { n.Policy.Restore(cfg.Network.Verify) }
