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
void wayland_cgo_cleanup();
*/
import "C"
import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"unsafe"
)

type WaylandCGODetector struct {
	timeout    time.Duration
	onIdle     func()
	onResume   func()
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
	initialized bool
}

// Global instance for CGO callbacks
var globalDetector *WaylandCGODetector

//export goIdleCallback
func goIdleCallback() {
	if globalDetector != nil && globalDetector.onIdle != nil {
		globalDetector.onIdle()
	}
}

//export goResumeCallback
func goResumeCallback() {
	if globalDetector != nil && globalDetector.onResume != nil {
		globalDetector.onResume()
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
	globalDetector = detector

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

	detector.initialized = true
	log.Println("Wayland CGO idle detector initialized successfully")

	return detector, nil
}

func (w *WaylandCGODetector) Start() error {
	if !w.initialized {
		return fmt.Errorf("detector not initialized")
	}

	// Run event loop in goroutine
	go func() {
		log.Println("Starting Wayland CGO event loop")
		for {
			select {
			case <-w.ctx.Done():
				log.Println("Wayland CGO event loop stopped")
				return
			default:
				ret := C.wayland_cgo_dispatch()
				if ret != 0 {
					log.Printf("Wayland dispatch error: %d", ret)
					return
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

	if w.initialized {
		C.wayland_cgo_cleanup()
		w.initialized = false
	}

	// Clear global detector
	globalDetector = nil
}

// Keep the compiler from complaining about unused imports
var _ = unsafe.Pointer(nil)
