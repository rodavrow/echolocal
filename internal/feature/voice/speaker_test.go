package voice

import (
	sharedavatar "github.com/ygelfand/libcountertop/pkg/assistant/avatar"
)

// The avatar finds these by asserting the conversation's speaker, so a method the wrapper
// does not have quietly falls back to 16 kHz mono and a guessed queue time.
var (
	_ sharedavatar.NativeSink = voiceSpeaker{}
	_ sharedavatar.TimedSink  = voiceSpeaker{}
	_ sharedavatar.Drainer    = voiceSpeaker{}
)
