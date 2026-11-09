package platform

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestLockSerializesAndSupersedes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "generate.lock")
	relA, err := AcquireGenerateLock(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	results := make([]error, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rel, err := AcquireGenerateLock(context.Background(), path)
			if err == nil {
				defer rel()
			}
			results[i] = err
		}(i)
		time.Sleep(50 * time.Millisecond) // deterministic ticket order
	}
	time.Sleep(50 * time.Millisecond)
	relA()
	wg.Wait()
	if !errors.Is(results[0], ErrSuperseded) {
		t.Errorf("older waiter = %v, want ErrSuperseded", results[0])
	}
	if results[1] != nil {
		t.Errorf("newest waiter = %v, want nil", results[1])
	}
}

func TestLockUncontendedAndReacquirable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "generate.lock")
	for i := 0; i < 3; i++ {
		rel, err := AcquireGenerateLock(context.Background(), path)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		rel()
	}
}

func TestLockContextCancel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "generate.lock")
	rel, err := AcquireGenerateLock(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer rel()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := AcquireGenerateLock(ctx, path); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
}
