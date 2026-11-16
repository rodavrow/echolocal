//go:build !board_biscuit

// Package videoplay lets Home Assistant play a video from the local network on the screen.
package videoplay

import (
	"sync"

	"github.com/ygelfand/libcountertop/pkg/audio/resample"
	"github.com/ygelfand/libcountertop/pkg/display/video"
	"github.com/ygelfand/libcountertop/pkg/media/playback"
	sharedvideoplay "github.com/ygelfand/libcountertop/pkg/media/videoplay"
	sharedcomponent "github.com/ygelfand/libcountertop/pkg/runtime/component"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/feature/playout"
	"github.com/ygelfand/echolocal/internal/hardware/display"
	"github.com/ygelfand/echolocal/internal/hardware/speaker"
)

func init() {
	component.Register(sharedcomponent.Device, Get, component.Needs(board.Panel))
}

// Get leaves HTTP unset: the player's own client has no proxy and refuses to connect off the
// local network, which a client from fetch cannot be made to do.
var Get = sync.OnceValue(func() *sharedvideoplay.Player {
	return sharedvideoplay.New(sharedvideoplay.Options{
		Output: func(ended func()) playback.Output {
			o := playout.New()
			o.SetEnded(ended)
			return o
		},
		Target: func() playback.Target {
			return video.Target(display.Get(), component.Board().MaxFPS)
		},
		Resample: func(from int) func([]int16) []int16 {
			return resample.NewRational(from, speaker.Rate, speaker.Channels).Run
		},
	})
})
