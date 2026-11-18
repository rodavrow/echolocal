package chromecast

import "testing"

func TestRestartOnChangeLeavesTheEngineAloneWhenQuietnessHolds(t *testing.T) {
	restarts := 0
	follow := restartOnChange(true, func() { restarts++ })

	follow(true)
	follow(true)

	if restarts != 0 {
		t.Errorf("restarted %d times for changes that kept the engine quiet", restarts)
	}
}

func TestRestartOnChangeRestartsTheEngineEachTimeQuietnessFlips(t *testing.T) {
	restarts := 0
	follow := restartOnChange(false, func() { restarts++ })

	follow(true)
	follow(true)
	follow(false)

	if restarts != 2 {
		t.Errorf("restarted %d times, want 2: once to go quiet, once to go back", restarts)
	}
}
