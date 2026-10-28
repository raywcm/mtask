// Package broker owns task lifetime and client attachment policy, independently of UI lifetime.
package broker

import (
	"errors"
	"io"
	"sync"

	"github.com/raywcm/mtask/internal/protocol"
)

const (
	OutputChunkSize  = 32 * 1024
	OutputQueueDepth = 64 // At most 2 MiB of queued output per attachment.
)

var (
	ErrBusy   = errors.New("task already has a controlling client")
	ErrClosed = errors.New("broker output is closed")
)

// Subscription is a lease. A stale disconnect must never evict a replacement.
type Subscription struct {
	Output  <-chan []byte
	Notices <-chan protocol.NoticeMessage
	Done    <-chan struct{}
	queue   chan []byte
	notices chan protocol.NoticeMessage
	done    chan struct{}
	reason  protocol.ErrorMessage // Published before Done closes.
	Mode    string
	Initial bool
	serial  uint64
	modern  bool
}

func (s *Subscription) Reason() protocol.ErrorMessage { <-s.Done; return s.reason }

type Hub struct {
	mu          sync.Mutex
	members     map[*Subscription]struct{}
	owner       *Subscription
	serial      uint64
	closed      bool
	capture     bool
	pending     [][]byte
	pendingSize int
}

func (h *Hub) CaptureInitial()                { h.mu.Lock(); defer h.mu.Unlock(); h.capture = true }
func (h *Hub) Attach() (*Subscription, error) { return h.AttachMode(protocol.AttachExclusive, false) }
func (h *Hub) AttachMode(mode string, modern bool) (*Subscription, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, ErrClosed
	}
	if !protocol.ValidAttachMode(mode) {
		return nil, errors.New("invalid attachment mode")
	}
	if mode == protocol.AttachExclusive && h.owner != nil {
		return nil, ErrBusy
	}
	if mode == protocol.AttachTakeover {
		for s := range h.members {
			if s.Mode != protocol.AttachReadOnly {
				h.disconnect(s, "taken_over", "session control was taken over by another terminal")
			}
		}
	}
	q, notices, done := make(chan []byte, OutputQueueDepth), make(chan protocol.NoticeMessage, 8), make(chan struct{})
	h.serial++
	s := &Subscription{Output: q, Notices: notices, Done: done, queue: q, notices: notices, done: done, Mode: mode, serial: h.serial, modern: modern}
	for _, p := range h.pending {
		q <- p
	}
	s.Initial = len(h.pending) > 0
	h.pending = nil
	h.pendingSize = 0
	h.capture = false
	if h.members == nil {
		h.members = make(map[*Subscription]struct{})
	}
	h.members[s] = struct{}{}
	if h.owner == nil && mode != protocol.AttachReadOnly {
		h.owner = s
	}
	h.notifyRoles()
	return s, nil
}
func (h *Hub) CanInput(s *Subscription) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.members[s]
	return ok && s.Mode != protocol.AttachReadOnly
}
func (h *Hub) OwnsSize(s *Subscription) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return s != nil && h.owner == s
}
func (h *Hub) RequestRefresh(s *Subscription) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.owner == nil || h.owner == s || !h.owner.modern {
		return false
	}
	select {
	case h.owner.notices <- protocol.NoticeMessage{Type: "refresh"}:
	default:
	}
	return true
}
func (h *Hub) Detach(s *Subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.members[s]; ok {
		h.disconnect(s, "", "")
		h.notifyRoles()
	}
}

// Output stays opaque. A slow subscriber alone is disconnected; other clients
// receive the complete byte stream, never a stream with missing chunks.
func (h *Hub) Publish(p []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	if len(h.members) == 0 && h.capture {
		if h.pendingSize+len(p) > OutputChunkSize*OutputQueueDepth {
			h.pending = nil
			h.pendingSize = 0
			h.capture = false
			return
		}
		for len(p) > 0 {
			n := min(len(p), OutputChunkSize)
			last := len(h.pending) - 1
			if last >= 0 && len(h.pending[last]) < OutputChunkSize {
				n = min(n, OutputChunkSize-len(h.pending[last]))
				h.pending[last] = append(h.pending[last], p[:n]...)
			} else {
				h.pending = append(h.pending, append([]byte(nil), p[:n]...))
			}
			h.pendingSize += n
			p = p[n:]
		}
		return
	}
	changed := false
	for len(p) > 0 && len(h.members) > 0 {
		n := min(len(p), OutputChunkSize)
		chunk := append([]byte(nil), p[:n]...)
		p = p[n:]
		for s := range h.members {
			select {
			case s.queue <- chunk:
			default:
				h.disconnect(s, "slow_client", "output queue overflow")
				changed = true
			}
		}
	}
	if changed {
		h.notifyRoles()
	}
}
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for s := range h.members {
		h.disconnect(s, "", "")
	}
}
func (h *Hub) disconnect(s *Subscription, code, message string) {
	delete(h.members, s)
	s.reason = protocol.ErrorMessage{Code: code, Message: message}
	close(s.queue)
	close(s.done)
	if h.owner == s {
		h.owner = nil
		for next := range h.members {
			if next.Mode != protocol.AttachReadOnly && (h.owner == nil || next.serial < h.owner.serial) {
				h.owner = next
			}
		}
	}
}
func (h *Hub) notifyRoles() {
	// A blocked notice consumer is a slow client too. Repeat after removal so
	// every remaining client receives the final size owner, including promotion.
	for {
		changed := false
		for s := range h.members {
			if !s.modern {
				continue
			}
			select {
			case s.notices <- protocol.NoticeMessage{Type: "role", SizeOwner: h.owner == s}:
			default:
				h.disconnect(s, "slow_client", "notification queue overflow")
				changed = true
			}
		}
		if !changed {
			return
		}
	}
}
func Pump(r io.Reader, h *Hub) error {
	buf := make([]byte, OutputChunkSize)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			h.Publish(buf[:n])
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}
