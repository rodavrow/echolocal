//go:build board_athena

package mic

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ygelfand/libcountertop/pkg/audio/alsa"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
)

// The microphone ring holds 30 ms and cannot be made deeper, so nothing but reading may stand
// between two reads of it. Mixing and echo suppression run on the listener's side of this queue.
const queuedPeriods = 32

var errReaderStopped = errors.New("mic: the capture reader stopped")

type pcmDevice interface {
	Read(buf []byte) (int, error)
	Close() error
}

type chunk struct {
	ref, mics []byte
	err       error
}

type opener func() (ref, mics pcmDevice, err error)

type reader struct {
	open       opener
	free, full chan *chunk
	done       chan struct{}
	stopped    chan struct{}
	closing    sync.Once
}

func newReader(open opener, refBytes, micBytes int) (*reader, error) {
	ref, mics, err := open()
	if err != nil {
		return nil, err
	}

	r := &reader{
		open:    open,
		free:    make(chan *chunk, queuedPeriods),
		full:    make(chan *chunk, queuedPeriods),
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	for range queuedPeriods {
		r.free <- &chunk{ref: make([]byte, refBytes), mics: make([]byte, micBytes)}
	}
	safe.Go("capture reader", func() { r.run(ref, mics) })
	return r, nil
}

// A listener that falls a whole queue behind stalls the reader, and the ring then overruns as it
// did when the listener read it directly.
func (r *reader) run(ref, mics pcmDevice) {
	defer close(r.stopped)
	defer close(r.full)
	defer func() { closePair(ref, mics) }()

	for {
		var c *chunk
		select {
		case <-r.done:
			return
		case c = <-r.free:
		}

		c.err = readPair(ref, mics, c)
		if errors.Is(c.err, alsa.ErrOverrun) {
			// Reopened together, or the reference drifts against the microphones.
			closePair(ref, mics)
			if ref, mics, c.err = r.reopen(); c.err == nil {
				c.err = alsa.ErrOverrun
			}
		}

		failed := c.err != nil && !errors.Is(c.err, alsa.ErrOverrun)
		r.full <- c
		if failed {
			return
		}
	}
}

func (r *reader) reopen() (ref, mics pcmDevice, err error) {
	if ref, mics, err = r.open(); err != nil {
		return nil, nil, fmt.Errorf("mic: restarting the paired capture: %w", err)
	}
	return ref, mics, nil
}

// The caller hands the chunk back with release once it has read it.
func (r *reader) next() (*chunk, error) {
	c, ok := <-r.full
	if !ok {
		return nil, errReaderStopped
	}
	return c, nil
}

func (r *reader) release(c *chunk) { r.free <- c }

func (r *reader) Close() {
	r.closing.Do(func() { close(r.done) })
	<-r.stopped
}

func readPair(ref, mics pcmDevice, c *chunk) error {
	if err := readFull(ref, c.ref); err != nil {
		return err
	}
	return readFull(mics, c.mics)
}

func readFull(d pcmDevice, buf []byte) error {
	for got := 0; got < len(buf); {
		n, err := d.Read(buf[got:])
		if err != nil {
			return err
		}
		got += n
	}
	return nil
}

func closePair(ref, mics pcmDevice) {
	if ref != nil {
		_ = ref.Close()
	}
	if mics != nil {
		_ = mics.Close()
	}
}
