// wayland_cgo.go - Wayland idle detection using CGO and native C bindings
package idle

/*
#cgo pkg-config: wayland-client
#cgo CFLAGS: -I${SRCDIR}/wayland-protocols
#include <stdint.h>

// External C functions defined in wayland_idle.c
int wayland_cgo_init();
int wayland_cgo_register_timeout(uint32_t timeout_ms);
int wayland_cgo_dispatch();
int wayland_cgo_get_fd();
void wayland_cgo_cleanup();
*/
import "C"
import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

type WaylandCGODetector struct {
	timeout     time.Duration
	onIdle      func()
	onResume    func()
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	initialized atomic.Bool
	onLost      func()
}

// Global instance for CGO callbacks
var globalDetector *WaylandCGODetector
var globalDetectorMu sync.Mutex

// SetOnLost registers a callback for the event loop ending unexpectedly.
//
// The loop can stop for reasons that have nothing to do with shutdown — a
// compositor restart, a socket replacement, a protocol error. Once it does,
// there is no idle source and no compositor resume, and nothing in the
// process will ever notice on its own. The callback is how the owner finds
// out and re-arms its fallback.
func (w *WaylandCGODetector) SetOnLost(fn func()) {
	w.mu.Lock()
	w.onLost = fn
	w.mu.Unlock()
}

// onLost fires the loss callback at most once per unexpected exit.
func (w *WaylandCGODetector) reportLost() {
	w.mu.Lock()
	fn := w.onLost
	w.mu.Unlock()
	if fn != nil {
		fn()
	}
}

//export goIdleCallback
func goIdleCallback() {
	globalDetectorMu.Lock()
	detector := globalDetector
	globalDetectorMu.Unlock()
	if detector != nil && detector.onIdle != nil {
		detector.onIdle()
	}
}

//export goResumeCallback
func goResumeCallback() {
	globalDetectorMu.Lock()
	detector := globalDetector
	globalDetectorMu.Unlock()
	if detector != nil && detector.onResume != nil {
		detector.onResume()
	}
}

func NewWaylandCGODetector(timeout time.Duration, onIdle func(), onResume func()) (*WaylandCGODetector, error) {
	ctx, cancel := context.WithCancel(context.Background())

	detector := &WaylandCGODetector{
		timeout:  timeout,
		onIdle:   onIdle,
		onResume: onResume,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Set global instance for CGO callbacks
	globalDetectorMu.Lock()
	globalDetector = detector
	globalDetectorMu.Unlock()

	// Initialize Wayland connection
	ret := C.wayland_cgo_init()
	if ret != 0 {
		cancel()
		return nil, fmt.Errorf("failed to initialize Wayland: error code %d", ret)
	}

	// Register idle timeout
	timeoutMs := C.uint32_t(timeout.Milliseconds())
	ret = C.wayland_cgo_register_timeout(timeoutMs)
	if ret != 0 {
		C.wayland_cgo_cleanup()
		cancel()
		return nil, fmt.Errorf("failed to register timeout: error code %d", ret)
	}

	detector.initialized.Store(true)
	log.Println("Wayland CGO idle detector initialized successfully")

	return detector, nil
}

func (w *WaylandCGODetector) Start() error {
	if !w.initialized.Load() {
		return fmt.Errorf("detector not initialized")
	}

	// Get the Wayland file descriptor for polling
	fd := int(C.wayland_cgo_get_fd())
	if fd < 0 {
		return fmt.Errorf("failed to get Wayland FD")
	}

	log.Printf("Using Wayland FD: %d for polling", fd)

	// Run event loop in goroutine with proper polling
	go func() {
		log.Println("Starting Wayland CGO event loop")
		lastHeartbeat := time.Now()
		pollCount := 0

		// Report why the loop ended. Once it returns there is no idle source
		// and no resume source, so the caller has to know to fall back rather
		// than sit idle forever with the fallback timer disabled.
		defer func() {
			if w.ctx.Err() == nil {
				log.Println("Wayland event loop ended while the detector was still expected to be running")
				w.reportLost()
			}
		}()

		for {
			select {
			case <-w.ctx.Done():
				log.Println("Wayland CGO event loop stopped")
				return
			default:
				// Use unix.Poll instead of C poll
				pollFds := []unix.PollFd{
					{
						Fd:     int32(fd),
						Events: unix.POLLIN,
					},
				}

				// Poll with 100ms timeout to allow checking ctx
				n, err := unix.Poll(pollFds, 100)
				if err != nil {
					// EINTR is expected when signals are delivered (like Ctrl+C)
					if err == unix.EINTR {
						continue
					}
					log.Printf("Poll error: %v", err)
					return
				}

				if n > 0 {
					revents := pollFds[0].Revents

					// POLLHUP means the peer closed the socket. Without this
					// test the POLLIN check below simply fails, nothing is
					// dispatched, and poll returns immediately again — a tight
					// loop that burns a core and reports nothing.
					if revents&(unix.POLLHUP|unix.POLLERR|unix.POLLNVAL) != 0 {
						log.Printf("Wayland connection closed (revents=0x%x), stopping event loop", revents)
						return
					}

					if revents&unix.POLLIN != 0 {
						// Dispatch pending events
						dispatchRet := C.wayland_cgo_dispatch()
						if dispatchRet < 0 {
							log.Printf("Wayland dispatch error: %d", dispatchRet)
							return
						}
					}
				}

				// Heartbeat logging every 5 minutes
				pollCount++
				if time.Since(lastHeartbeat) >= 5*time.Minute {
					log.Printf("[Heartbeat] Wayland event loop active, polls: %d", pollCount)
					lastHeartbeat = time.Now()
					pollCount = 0
				}
			}
		}
	}()

	return nil
}

func (w *WaylandCGODetector) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.cancel()

	if w.initialized.Load() {
		C.wayland_cgo_cleanup()
		w.initialized.Store(false)
	}

	// Clear global detector
	globalDetectorMu.Lock()
	globalDetector = nil
	globalDetectorMu.Unlock()
}

// Keep the compiler from complaining about unused imports
var _ = unsafe.Pointer(nil)
