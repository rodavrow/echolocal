// Package announcements shows speech Home Assistant pushes to the device as a reply.
package announcements

import (
	"sync"

	sharedannouncements "github.com/ygelfand/libcountertop/pkg/assistant/announcements"
	sharedcomponent "github.com/ygelfand/libcountertop/pkg/runtime/component"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/avatar"
	"github.com/ygelfand/echolocal/internal/feature/wakeword"
)

func init() {
	component.Register(sharedcomponent.Device, Get, sharedcomponent.Order(25))
}

var Get = sync.OnceValue(func() *sharedannouncements.Announcements {
	a := sharedannouncements.New(sharedannouncements.Options{
		Slots: wakeword.Slots,
		Read:  func() bool { return config.Get().Wake.AnimateAnnouncements },
		Save:  func(on bool) error { return config.Set().Announcements().Animate(on) },
		// Opening the avatar early hides the second LiveTalking needs to start.
		Chosen: avatar.Chosen,
	})
	component.Settings.Add(a.Controls())
	return a
})
