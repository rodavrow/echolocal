package chromecast

import (
	"sync"

	networkpolicy "github.com/ygelfand/echolocal/internal/feature/network"
)

// quiet reports whether the Internet access setting keeps cast from reaching Google: the engine's
// own background fetches and the lookups a stock Cast device makes.
func quiet() bool { return networkpolicy.Get().Internet().Quiet() }

// restartOnChange returns a func to call with the setting's current quietness. It calls restart
// only when that differs from the last value, starting from was: the policy announces every
// network change, and restarting the engine closes whatever it is showing.
func restartOnChange(was bool, restart func()) func(now bool) {
	var mu sync.Mutex
	return func(now bool) {
		mu.Lock()
		changed := now != was
		was = now
		mu.Unlock()
		if changed {
			restart()
		}
	}
}
