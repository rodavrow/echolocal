// Package captions switches the text of what an assistant heard and replies on the screen.
package captions

import (
	"sync"

	sharedcaptions "github.com/ygelfand/libcountertop/pkg/assistant/captions"
	sharedcomponent "github.com/ygelfand/libcountertop/pkg/runtime/component"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
)

func init() {
	component.Register(sharedcomponent.Device, Get, sharedcomponent.Order(25))
}

var Get = sync.OnceValue(func() *sharedcaptions.Captions {
	c := sharedcaptions.New(sharedcaptions.Options{
		Read: func() bool { return config.Get().Wake.ShowCaptions },
		Save: func(on bool) error { return config.Set().Captions().Show(on) },
	})
	component.Settings.Add(c.Controls())
	return c
})
