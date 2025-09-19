package browser

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grandcat/zeroconf"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

func TestRun_GoroutineLeakOnContextCancel(t *testing.T) {
	// This test reproduces the deadlock/leak scenario.
	// The mock resolver will block until the context is canceled but will NOT
	// close the channel, which is the behavior of the real library.
	// This causes the fan-in goroutine to leak, and wg.Wait() to deadlock.

	mockImpl := &mockZeroconf{
		resolver: &mockResolver{
			browseShouldBlock: true,
		},
	}

	session := NewZeroconfSession(mockImpl, nil, "_test._tcp", "local.")
	fanInCh := make(chan *zeroconf.ServiceEntry)
	ctx, cancel := context.WithCancel(context.Background())

	runFinished := make(chan struct{})
	go func() {
		session.Run(ctx, fanInCh)
		close(runFinished)
	}()

	// Give the session a moment to start up.
	time.Sleep(50 * time.Millisecond)

	// Cancel the context, which should signal the session to shut down.
	cancel()

	// Wait for the Run function to finish. If it deadlocks, this will time out.
	select {
	case <-runFinished:
		// This is the desired outcome for the *fixed* code.
		// For the *original* (buggy) code, this case will not be reached.
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Test timed out: session.Run() deadlocked and did not terminate after context cancellation.")
	}
}

func TestRun_DoesNotCloseInputChannel(t *testing.T) {
	// This test ensures that the Run method does not close the fanInCh
	// channel that is passed into it. The caller who creates the channel
	// is responsible for closing it.
	mockImpl := &mockZeroconf{
		resolver: &mockResolver{
			// Configure the mock to not block, so Run returns immediately.
			browseShouldBlock: false,
		},
	}

	session := NewZeroconfSession(mockImpl, nil, "_test._tcp", "local.")
	fanInCh := make(chan *zeroconf.ServiceEntry, 1)
	ctx, cancel := context.WithCancel(context.Background())

	// Run the session. It should complete without errors and without closing fanInCh.
	if err := session.Run(ctx, fanInCh); err != nil {
		t.Fatalf("session.Run returned an unexpected error: %v", err)
	}

	cancel()
	<-ctx.Done()
	time.Sleep(1) // just wait a sec for chan to close.

	// The assertion: try to close the channel. If Run() already closed it,
	// this will cause a panic, and the test will fail.
	// We wrap this in a recover to provide a more helpful error message.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Test panicked because fanInCh was closed by session.Run, but it should not have been. Panic: %v", r)
		}
	}()
	close(fanInCh)
}

func TestSessionStartup(t *testing.T) {
	clog.D.Set()

	b := NewZeroconfBrowser(".local", "_type", nil)
	testCases := []struct {
		tcase         string
		expectedError string
		zeroconfImpl  ZeroconfInterface
	}{
		{"queryService succeeds", "", mockZeroconf{  resolver: &mockResolver{} }},
		{"NewResolver fails", "test resolver error", mockZeroconf{ newResolverShouldError: true, resolver: &mockResolver{}, }},
		{"Browse fails", "test browse error", mockZeroconf{ resolver: &mockResolver{ browseShouldError: true, }, }},
	}
	for _, tc := range testCases {
		t.Logf("Starting test: %s\n", tc.tcase)
		session := NewZeroconfSession(tc.zeroconfImpl, b.interfaces, b.mdnsType, b.domain)
		entriesCh := make(chan *zeroconf.ServiceEntry)
		ctx, cancel := context.WithCancel(context.Background())
		result := session.Run(ctx, entriesCh)
		cancel()
		<- ctx.Done()
		if tc.expectedError == "" {
			if result != nil {
				t.Errorf("Unexpected failure in %v: %v", tc.tcase, result)
			}
		} else {
			if result.Error() != tc.expectedError {
				t.Errorf("Unexpected result in %v: %v, but expected %v", tc.tcase, result, tc.expectedError)
			}
		}
	}
}



// mockResolver implements the zeroconf.Resolver interface for testing.
type mockResolver struct {
	browseShouldError bool
	browseShouldBlock bool
}

func (m *mockResolver) Browse(ctx context.Context, service, domain string, entries chan<- *zeroconf.ServiceEntry) error {
	defer close(entries)

	if m.browseShouldError {
		return errors.New("test browse error")
	}
	if m.browseShouldBlock {
		// This simulates the real behavior of blocking until the context is done.
		// Crucially, it does NOT close the entries channel, which is the
		// behavior that can lead to a leak in the original code.
		<-ctx.Done()
	}

	return ctx.Err()
}

func (m *mockResolver) Lookup(ctx context.Context, instance, service, domain string) (*zeroconf.ServiceEntry, error) {
	return nil, nil // Not needed for this test
}

func (m *mockResolver) Shutdown() {
	// Not needed for this test
}

// mockZeroconf implements ZeroconfInterface for testing.
type mockZeroconf struct {
	resolver          *mockResolver
	newResolverShouldError bool
}

func (m mockZeroconf) NewResolver(opts ...zeroconf.ClientOption) (ResolverInterface, error) {
	if m.newResolverShouldError {
		return nil, errors.New("test resolver error")
	}
	return m.resolver, nil
}
