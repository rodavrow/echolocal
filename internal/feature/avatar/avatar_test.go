//go:build !board_biscuit

package avatar

import (
	"slices"
	"testing"

	esphome "github.com/ygelfand/go-esphome-device"

	sharedavatar "github.com/ygelfand/libcountertop/pkg/assistant/avatar"
	"github.com/ygelfand/libcountertop/pkg/display/callview"
	"github.com/ygelfand/libcountertop/pkg/display/cameraview"
	"github.com/ygelfand/libcountertop/pkg/display/doorbellview"
	"github.com/ygelfand/libcountertop/pkg/display/idleview"
	sharedshell "github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/videopage"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/feature/wakeword"
)

func TestEveryVideoViewTakesTheScreenFromTheAvatar(t *testing.T) {
	for _, v := range []sharedshell.View{
		&videopage.Page{}, &cameraview.Live{}, &cameraview.Grid{}, &doorbellview.Live{},
		&callview.Talk{},
	} {
		if !showsVideo(v) {
			t.Errorf("%T does not take the screen from the avatar", v)
		}
	}
}

func TestThePhotoScreenLeavesTheAvatarAlone(t *testing.T) {
	if showsVideo(&idleview.View{}) {
		t.Error("the photo screen hides the avatar")
	}
}

// Voice Assistant Manager finds these by display name on the assistant's sub-device.
var names = []string{
	"Avatar mode", "Avatar URL", "Avatar ID", "Avatar connection", "Avatar session",
	"Avatar state",
}

func TestEachAssistantHasItsAvatarEntitiesOnItsOwnDevice(t *testing.T) {
	a := &Avatar{sharedavatar.New(sharedavatar.Options{
		Slots:    wakeword.Slots,
		DeviceID: component.AssistantDevice,
	})}

	got := map[uint32][]string{}
	for _, e := range a.Entities() {
		b := base(t, e)
		got[b.DeviceID] = append(got[b.DeviceID], b.Name)
	}

	for slot := range wakeword.Slots {
		if on := got[component.AssistantDevice(slot)]; !slices.Equal(on, names) {
			t.Errorf("assistant %d has %v, want %v", slot+1, on, names)
		}
	}
	if len(got) != wakeword.Slots+1 {
		t.Errorf("avatar entities on %d devices, want %d", len(got), wakeword.Slots+1)
	}
}

// Voice Assistant Manager finds the switch by this name on the device that carries the
// satellite, which is the main one.
func TestTheShowItselfHasTheAvatarHoldSwitch(t *testing.T) {
	a := &Avatar{sharedavatar.New(sharedavatar.Options{
		Slots:    wakeword.Slots,
		DeviceID: component.AssistantDevice,
	})}

	var onShow []string
	for _, e := range a.Entities() {
		if b := base(t, e); b.DeviceID == 0 {
			onShow = append(onShow, b.Name)
		}
	}

	if !slices.Equal(onShow, []string{"Avatar hold"}) {
		t.Errorf("the Show itself has %v, want only Avatar hold", onShow)
	}
}

func base(t *testing.T, e esphome.Entity) *esphome.Base {
	t.Helper()
	switch e := e.(type) {
	case *esphome.Switch:
		return &e.Base
	case *esphome.Text:
		return &e.Base
	case *esphome.Select:
		return &e.Base
	case *esphome.TextSensor:
		return &e.Base
	}
	t.Fatalf("unexpected avatar entity %T", e)
	return nil
}
