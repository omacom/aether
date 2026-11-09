package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ErrSuperseded is returned by AcquireGenerateLock when a newer request
// arrived while this one waited; the caller should exit without applying.
var ErrSuperseded = errors.New("superseded by a newer request")

// GenerateLockPath returns ~/.local/state/aether/generate.lock.
func GenerateLockPath() string {
	return filepath.Join(homeDir(), ".local", "state", "aether", "generate.lock")
}

func flockFile(ctx context.Context, f *os.File) error {
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return nil
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EINTR {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// nextTicket atomically increments the sequence counter stored at seqPath.
func nextTicket(ctx context.Context, seqPath string) (int, error) {
	f, err := os.OpenFile(seqPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if err := flockFile(ctx, f); err != nil {
		return 0, err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	n := readSeq(f) + 1
	if err := f.Truncate(0); err != nil {
		return 0, err
	}
	if _, err := f.WriteAt([]byte(strconv.Itoa(n)), 0); err != nil {
		return 0, err
	}
	return n, nil
}

func readSeq(f *os.File) int {
	buf := make([]byte, 32)
	n, _ := f.ReadAt(buf, 0)
	v, _ := strconv.Atoi(strings.TrimSpace(string(buf[:n])))
	return v
}

// AcquireGenerateLock serialises generate runs with flock(LOCK_EX) on path.
// Each call first takes a ticket from a sequence file (path + ".seq"). Once a
// call holds the lock it checks the ticket: if a newer request has taken a
// later ticket, it releases the lock and returns ErrSuperseded, so only the
// newest waiter applies (last wins).
func AcquireGenerateLock(ctx context.Context, path string) (release func(), err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	seqPath := path + ".seq"
	ticket, err := nextTicket(ctx, seqPath)
	if err != nil {
		return nil, fmt.Errorf("sequence file: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	if err := flockFile(ctx, f); err != nil {
		f.Close()
		return nil, err
	}
	release = func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}
	sf, err := os.Open(seqPath)
	if err == nil {
		latest := readSeq(sf)
		sf.Close()
		if latest > ticket {
			release()
			return nil, ErrSuperseded
		}
	}
	return release, nil
}
