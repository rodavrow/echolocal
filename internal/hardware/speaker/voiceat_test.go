package speaker

import (
	"slices"
	"testing"
	"time"

	"github.com/ygelfand/echolocal/internal/config"
)

func TestStereoAtTheSpeakerRateIsQueuedAsItIs(t *testing.T) {
	in := []int16{1, 2, 3, 4}

	got, err := asOutput(Rate, Channels, in)

	if err != nil || !slices.Equal(got, in) {
		t.Errorf("asOutput = %v, %v, want %v", got, err, in)
	}
}

func TestMonoAtTheSpeakerRateGoesToBothChannels(t *testing.T) {
	got, err := asOutput(Rate, 1, []int16{7, -7})

	if want := []int16{7, 7, -7, -7}; err != nil || !slices.Equal(got, want) {
		t.Errorf("asOutput = %v, %v, want %v", got, err, want)
	}
}

func TestVoiceAtAnotherRateIsRefused(t *testing.T) {
	if _, err := asOutput(44100, Channels, []int16{1, 2}); err == nil {
		t.Error("44.1 kHz voice was taken as 48 kHz")
	}
}

func TestVoiceInMoreChannelsIsRefused(t *testing.T) {
	if _, err := asOutput(Rate, 6, make([]int16, 6)); err == nil {
		t.Error("six channels were taken as stereo")
	}
}

func TestAVoiceLaneTimesOnlyItsOwnQueue(t *testing.T) {
	p := newTestPlayer()
	p.bus.Push(config.StreamVoice, make([]int16, Rate/10*Channels))
	p.bus.Push(config.StreamMedia, make([]int16, Rate*Channels))

	got := p.Lane(config.StreamVoice).QueuedFor()

	if want := 100 * time.Millisecond; got != want {
		t.Errorf("QueuedFor = %v, want %v", got, want)
	}
}

func TestAnEmptyLaneWaitsForNothing(t *testing.T) {
	if got := newTestPlayer().Lane(config.StreamVoice).QueuedFor(); got != 0 {
		t.Errorf("QueuedFor = %v, want 0", got)
	}
}
