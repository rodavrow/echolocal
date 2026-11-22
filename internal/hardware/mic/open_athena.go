//go:build board_athena

package mic

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/ygelfand/libcountertop/pkg/audio/alsa"
	"github.com/ygelfand/libcountertop/pkg/audio/resample"
)

const (
	micDevice   = 17
	micChannels = 4
	refDevice   = 9
	refChannels = 2

	inputRate   = 96000
	inputPeriod = inputRate / 100

	// Device 17's DMA block is 48 KB: a 64 KB ring faulted the kernel 48 KB in.
	micPeriods = 3
	refPeriods = 4

	// AFE.cfg MicBoostdB.
	micBoostDB = 5.89
)

type paired struct {
	in           *reader
	micRS, refRS *resample.Rational

	micIn, refIn   []int16
	micOut, refOut []int16
}

const (
	refPeriodBytes = inputPeriod * refChannels * 2
	micPeriodBytes = inputPeriod * micChannels * 4
)

func openCapture() (stream, error) {
	setupConverter()

	in, err := newReader(openPair, refPeriodBytes, micPeriodBytes)
	if err != nil {
		return nil, err
	}
	return newPaired(in), nil
}

func newPaired(in *reader) *paired {
	p := &paired{
		in:    in,
		micRS: resample.NewRational(inputRate, Rate, micChannels),
		refRS: resample.NewRational(inputRate, Rate, refChannels),
		micIn: make([]int16, inputPeriod*micChannels),
		refIn: make([]int16, inputPeriod*refChannels),
	}
	p.micRS.Headroom(float32(math.Pow(10, micBoostDB/20)))
	return p
}

func openPair() (ref, mics pcmDevice, err error) {
	r, err := alsa.Open(Card, refDevice, alsa.Config{
		Channels:   refChannels,
		Rate:       inputRate,
		Format:     alsa.FormatS16LE,
		Bits:       16,
		PeriodSize: inputPeriod,
		Periods:    refPeriods,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("reference: %w", err)
	}

	m, err := alsa.Open(Card, micDevice, alsa.Config{
		Channels:   micChannels,
		Rate:       inputRate,
		Format:     alsa.FormatS32LE,
		Bits:       32,
		PeriodSize: inputPeriod,
		Periods:    micPeriods,
	})
	if err != nil {
		_ = r.Close()
		return nil, nil, fmt.Errorf("microphones: %w", err)
	}
	return r, m, nil
}

// What was resampled before an overrun does not join up with what follows it.
func (p *paired) reset() {
	p.micRS.Reset()
	p.refRS.Reset()
	p.micOut, p.refOut = p.micOut[:0], p.refOut[:0]
}

func (p *paired) Read(buf []byte) (int, error) {
	const frameBytes = Channels * Bits / 8

	frames := len(buf) / frameBytes
	for len(p.micOut) < frames*micChannels || len(p.refOut) < frames*refChannels {
		if err := p.fill(); err != nil {
			return 0, err
		}
	}

	for f := range frames {
		out := buf[f*frameBytes:]
		for c := range micChannels {
			putS24(out[3*c:], p.micOut[f*micChannels+c])
		}
		for c := range refChannels {
			putS24(out[3*(Mics+c):], p.refOut[f*refChannels+c])
		}
	}

	p.micOut = append(p.micOut[:0], p.micOut[frames*micChannels:]...)
	p.refOut = append(p.refOut[:0], p.refOut[frames*refChannels:]...)
	return frames * frameBytes, nil
}

func (p *paired) fill() error {
	got, err := p.in.next()
	if err != nil {
		return err
	}
	defer p.in.release(got)

	if got.err != nil {
		if errors.Is(got.err, alsa.ErrOverrun) {
			p.reset()
		}
		return got.err
	}

	for i := range p.refIn {
		p.refIn[i] = int16(binary.LittleEndian.Uint16(got.ref[2*i:]))
	}
	// Measured on athena: 24-bit samples in the low bits of each 32-bit slot.
	for i := range p.micIn {
		p.micIn[i] = int16(int32(binary.LittleEndian.Uint32(got.mics[4*i:])) >> 8)
	}

	p.micOut = append(p.micOut, p.micRS.Run(p.micIn)...)
	p.refOut = append(p.refOut, p.refRS.Run(p.refIn)...)
	return nil
}

func (p *paired) Close() error {
	p.in.Close()
	return nil
}

func putS24(b []byte, v int16) {
	s := int32(v) << 8
	b[0], b[1], b[2] = byte(s), byte(s>>8), byte(s>>16)
}
