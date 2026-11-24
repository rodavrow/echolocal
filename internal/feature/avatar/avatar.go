//go:build !board_biscuit

// Package avatar has an assistant in avatar mode shown and voiced by its LiveTalking avatar.
package avatar

import (
	"context"
	"slices"
	"sync"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	sharedavatar "github.com/ygelfand/libcountertop/pkg/assistant/avatar"
	"github.com/ygelfand/libcountertop/pkg/assistant/turn"
	sharedscreen "github.com/ygelfand/libcountertop/pkg/display/assistant"
	"github.com/ygelfand/libcountertop/pkg/display/callview"
	"github.com/ygelfand/libcountertop/pkg/display/cameraview"
	"github.com/ygelfand/libcountertop/pkg/display/doorbellview"
	sharedshell "github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/videopage"
	"github.com/ygelfand/libcountertop/pkg/fetch"
	sharedcomponent "github.com/ygelfand/libcountertop/pkg/runtime/component"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/shell"
	"github.com/ygelfand/echolocal/internal/feature/wakeword"
	"github.com/ygelfand/echolocal/internal/hardware/display"
)

// clipWait bounds one LiveTalking request or reply download.
const clipWait = 30 * time.Second

// The assistant screen finds the avatar's placement through the conversation's avatar.
var _ sharedscreen.AvatarScreen = (*sharedavatar.Avatars)(nil)

func init() {
	component.Register(
		sharedcomponent.Network,
		Get,
		sharedcomponent.Order(30),
		component.Needs(board.Panel),
	)
}

type Avatar struct{ *sharedavatar.Avatars }

var Get = sync.OnceValue(func() *Avatar {
	return &Avatar{sharedavatar.New(sharedavatar.Options{
		Slots:     wakeword.Slots,
		Read:      func() schema.Wake { return config.Get().Wake },
		Write:     func(slot int) schema.WakeWriter { return config.Set().Wake(slot) },
		DeviceID:  component.AssistantDevice,
		Client:    fetch.Client(clipWait),
		Display:   display.Get(),
		VideoBusy: videoBusy,
	})}
})

// Turn is the avatar a conversation speaks through; nil on a board without a screen.
func Turn() turn.Avatar {
	if !component.Board().Has(board.Panel) {
		return nil
	}
	return Get().Avatars
}

// Chosen opens slot's avatar early when Home Assistant names it for the next announcement,
// which hides the second LiveTalking needs to start.
func Chosen(slot int, on bool) {
	if component.Board().Has(board.Panel) {
		Get().Chosen(slot, on)
	}
}

// videoBusy reports that a view playing video owns the hardware video, so the assistant shows
// its usual look. Views that only cover the home screen, such as the photo screen or the music
// app, leave the avatar alone.
func videoBusy() bool { return slices.ContainsFunc(shell.Get().Views(), showsVideo) }

// showsVideo is true for the views that play video. Cast, AirPlay, YouTube and local video all
// play in a video page, and a call holds its talk view on the shell for as long as it lasts.
func showsVideo(v sharedshell.View) bool {
	switch v.(type) {
	case *videopage.Page, *cameraview.Live, *cameraview.Grid, *doorbellview.Live,
		*callview.Talk:
		return true
	}
	return false
}

func (a *Avatar) Run(ctx context.Context) error {
	a.Avatars.Run(ctx)
	return nil
}

func (a *Avatar) Entities() []esphome.Entity {
	var out []esphome.Entity
	for slot := range wakeword.Slots {
		out = append(out, a.Avatars.Entities(slot)...)
	}
	return out
}

func (a *Avatar) Restore(c config.Config) { a.Avatars.Restore(c.Wake.Slots(wakeword.Slots)) }
