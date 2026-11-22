//go:build board_athena

package mic

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/audio/alsa"
)

const (
	testRefBytes = 4
	testMicBytes = 8
)

// A device that never blocks: each read fills the buffer with the read's number.
type fakeCapture struct {
	reads  atomic.Int64
	closed atomic.Bool
	read   chan struct{}

	mu    sync.Mutex
	fails map[int64]error
}

func newFakeCapture() *fakeCapture {
	return &fakeCapture{read: make(chan struct{}, 4*queuedPeriods), fails: map[int64]error{}}
}

func (f *fakeCapture) failAt(nth int64, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fails[nth] = err
}

func (f *fakeCapture) Read(buf []byte) (int, error) {
	nth := f.reads.Add(1)
	f.mu.Lock()
	err := f.fails[nth]
	f.mu.Unlock()
	if err != nil {
		return 0, err
	}
	for i := range buf {
		buf[i] = byte(nth)
	}
	select {
	case f.read <- struct{}{}:
	default:
	}
	return len(buf), nil
}

func (f *fakeCapture) Close() error {
	f.closed.Store(true)
	return nil
}

type fakePair struct{ ref, mics *fakeCapture }

// Hands out the prepared pairs in order, then fails.
type fakeOpener struct {
	mu    sync.Mutex
	pairs []fakePair
	opens int
}

func (o *fakeOpener) open() (ref, mics pcmDevice, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.opens >= len(o.pairs) {
		o.opens++
		return nil, nil, errors.New("no device")
	}
	p := o.pairs[o.opens]
	o.opens++
	return p.ref, p.mics, nil
}

func (o *fakeOpener) opened() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.opens
}

func newFakePair() fakePair { return fakePair{ref: newFakeCapture(), mics: newFakeCapture()} }

func startReader(t *testing.T, pairs ...fakePair) (*reader, *fakeOpener) {
	t.Helper()
	o := &fakeOpener{pairs: pairs}
	r, err := newReader(o.open, testRefBytes, testMicBytes)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.Close)
	return r, o
}

func take(t *testing.T, r *reader) *chunk {
	t.Helper()
	c, err := r.next()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestReaderHandsOverPeriodsInOrder(t *testing.T) {
	r, _ := startReader(t, newFakePair())

	for nth := byte(1); nth <= 3; nth++ {
		c := take(t, r)
		if c.err != nil {
			t.Fatal(c.err)
		}
		if len(c.ref) != testRefBytes || len(c.mics) != testMicBytes {
			t.Fatalf("period %d is %d and %d bytes", nth, len(c.ref), len(c.mics))
		}
		if c.ref[0] != nth || c.mics[testMicBytes-1] != nth {
			t.Fatalf("period %d carries reads %d and %d", nth, c.ref[0], c.mics[testMicBytes-1])
		}
		r.release(c)
	}
}

func TestRegressionReaderKeepsReadingWhileNothingListens(t *testing.T) {
	pair := newFakePair()
	startReader(t, pair)

	for range queuedPeriods {
		<-pair.mics.read
	}

	if got := pair.mics.reads.Load(); got < queuedPeriods {
		t.Fatalf("the reader read %d periods with nobody listening, want %d", got, queuedPeriods)
	}
}

func TestReaderReopensBothDevicesAfterAnOverrun(t *testing.T) {
	first, second := newFakePair(), newFakePair()
	first.mics.failAt(2, alsa.ErrOverrun)
	r, o := startReader(t, first, second)

	r.release(take(t, r))
	c := take(t, r)
	if !errors.Is(c.err, alsa.ErrOverrun) {
		t.Fatalf("the second period reports %v, want an overrun", c.err)
	}
	r.release(c)

	c = take(t, r)
	if c.err != nil {
		t.Fatal(c.err)
	}
	if !first.ref.closed.Load() || !first.mics.closed.Load() {
		t.Error("the overrun left a device of the first pair open")
	}
	if o.opened() != 2 {
		t.Errorf("the devices were opened %d times, want 2", o.opened())
	}
	if c.mics[0] != 1 {
		t.Errorf("the period after the overrun is read %d of its device, want 1", c.mics[0])
	}
}

func TestReaderStopsWhenTheDevicesCannotBeReopened(t *testing.T) {
	only := newFakePair()
	only.ref.failAt(1, alsa.ErrOverrun)
	r, _ := startReader(t, only)

	c := take(t, r)
	if c.err == nil || errors.Is(c.err, alsa.ErrOverrun) {
		t.Fatalf("a failed reopen reports %v, want the reopen error", c.err)
	}
	if _, err := r.next(); !errors.Is(err, errReaderStopped) {
		t.Fatalf("after a failed reopen next reports %v, want %v", err, errReaderStopped)
	}
}

func TestReaderStopsOnADeviceError(t *testing.T) {
	broken := errors.New("device gone")
	pair := newFakePair()
	pair.mics.failAt(1, broken)
	r, _ := startReader(t, pair)

	if c := take(t, r); !errors.Is(c.err, broken) {
		t.Fatalf("the period reports %v, want %v", c.err, broken)
	}
	if _, err := r.next(); !errors.Is(err, errReaderStopped) {
		t.Fatalf("after a device error next reports %v, want %v", err, errReaderStopped)
	}
	if !pair.ref.closed.Load() || !pair.mics.closed.Load() {
		t.Error("the stopped reader left a device open")
	}
}

func TestReaderCloseClosesTheDevices(t *testing.T) {
	pair := newFakePair()
	r, _ := startReader(t, pair)

	r.Close()

	if !pair.ref.closed.Load() || !pair.mics.closed.Load() {
		t.Error("Close left a device open")
	}
}

func TestNewReaderReportsDevicesThatDoNotOpen(t *testing.T) {
	o := &fakeOpener{}

	if _, err := newReader(o.open, testRefBytes, testMicBytes); err == nil {
		t.Fatal("newReader opened without devices")
	}
}

func TestRegressionPairedReadCarriesOnAfterAnOverrun(t *testing.T) {
	first, second := newFakePair(), newFakePair()
	first.mics.failAt(2, alsa.ErrOverrun)
	o := &fakeOpener{pairs: []fakePair{first, second}}
	in, err := newReader(o.open, refPeriodBytes, micPeriodBytes)
	if err != nil {
		t.Fatal(err)
	}
	p := newPaired(in)
	t.Cleanup(func() { _ = p.Close() })
	frame := make([]byte, FrameSamples*Channels*Bits/8)

	_, err = p.Read(frame)

	if !errors.Is(err, alsa.ErrOverrun) {
		t.Fatalf("the read across the overrun reports %v, want an overrun", err)
	}
	if len(p.micOut) != 0 || len(p.refOut) != 0 {
		t.Errorf("the overrun left %d and %d resampled samples", len(p.micOut), len(p.refOut))
	}
	if n, err := p.Read(frame); err != nil || n != len(frame) {
		t.Fatalf("the read after the overrun gave %d bytes and %v, want %d", n, err, len(frame))
	}
}
