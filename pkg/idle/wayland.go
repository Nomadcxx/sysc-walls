// wayland.go - Wayland idle detection using ext-idle-notify-v1
package idle

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/rajveermalviya/go-wayland/wayland/client"
	ext_idle_notify "github.com/rajveermalviya/go-wayland/wayland/staging/ext-idle-notify-v1"
)

type WaylandIdleDetector struct {
	display       *client.Display
	registry      *client.Registry
	idleNotifier  *ext_idle_notify.IdleNotifier
	seat          *client.Seat
	notification  *ext_idle_notify.IdleNotification
	timeout       time.Duration
	onIdle        func()
	onResume      func()
	mu            sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewWaylandIdleDetector(timeout time.Duration, onIdle func(), onResume func()) (*WaylandIdleDetector, error) {
	// Connect to Wayland display
	display, err := client.Connect("")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Wayland display: %w", err)
	}

	registry, err := display.GetRegistry()
	if err != nil {
		display.Context().Close()
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	detector := &WaylandIdleDetector{
		display:  display,
		registry: registry,
		timeout:  timeout,
		onIdle:   onIdle,
		onResume: onResume,
		ctx:      ctx,
		cancel:   cancel,
	}

	if err := detector.initialize(); err != nil {
		display.Context().Close()
		cancel()
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	return detector, nil
}

func (w *WaylandIdleDetector) initialize() error {
	var idleNotifierName, idleNotifierVersion uint32
	var seatName, seatVersion uint32

	// Register handlers for global objects
	w.registry.SetGlobalHandler(func(e client.RegistryGlobalEvent) {
		switch e.Interface {
		case "ext_idle_notifier_v1":
			idleNotifierName = e.Name
			idleNotifierVersion = e.Version
		case "wl_seat":
			if seatName == 0 { // Only bind the first seat
				seatName = e.Name
				seatVersion = e.Version
			}
		}
	})

	// Perform roundtrips to ensure all globals are discovered
	w.displayRoundtrip()
	w.displayRoundtrip()

	// Check if we found required interfaces
	if idleNotifierName == 0 {
		return fmt.Errorf("compositor does not support ext_idle_notifier_v1 protocol")
	}
	if seatName == 0 {
		return fmt.Errorf("no seat found")
	}

	// Bind to the idle notifier
	w.idleNotifier = ext_idle_notify.NewIdleNotifier(w.display.Context())
	if err := w.registry.Bind(idleNotifierName, "ext_idle_notifier_v1", idleNotifierVersion, w.idleNotifier); err != nil {
		return fmt.Errorf("failed to bind idle notifier: %w", err)
	}

	// Bind to the seat
	w.seat = client.NewSeat(w.display.Context())
	if err := w.registry.Bind(seatName, "wl_seat", seatVersion, w.seat); err != nil {
		return fmt.Errorf("failed to bind seat: %w", err)
	}

	// Register idle timeout
	if err := w.registerIdleTimeout(); err != nil {
		return fmt.Errorf("failed to register idle timeout: %w", err)
	}

	return nil
}

func (w *WaylandIdleDetector) displayRoundtrip() error {
	callback, err := w.display.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}
	defer callback.Destroy()

	done := false
	callback.SetDoneHandler(func(_ client.CallbackDoneEvent) {
		done = true
	})

	// Dispatch events until done
	for !done {
		if err := w.display.Context().Dispatch(); err != nil {
			return fmt.Errorf("dispatch error: %w", err)
		}
	}

	return nil
}

func (w *WaylandIdleDetector) registerIdleTimeout() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Convert timeout to milliseconds
	timeoutMs := uint32(w.timeout.Milliseconds())

	// Get idle notification - this monitors actual input (keyboard, mouse, touch)
	// NOT idle inhibitors (which would be GetIdleNotification)
	notification, err := w.idleNotifier.GetIdleNotification(timeoutMs, w.seat)
	if err != nil {
		return fmt.Errorf("failed to get idle notification: %w", err)
	}

	// Set handler for when system goes idle
	notification.SetIdledHandler(func(e ext_idle_notify.IdleNotificationIdledEvent) {
		log.Println("Wayland idle detected")
		if w.onIdle != nil {
			w.onIdle()
		}
	})

	// Set handler for when system resumes from idle
	notification.SetResumedHandler(func(e ext_idle_notify.IdleNotificationResumedEvent) {
		log.Println("Wayland activity detected (resumed)")
		if w.onResume != nil {
			w.onResume()
		}
	})

	w.notification = notification
	return nil
}

func (w *WaylandIdleDetector) Start() error {
	// Run the event loop
	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			default:
				if err := w.display.Context().Dispatch(); err != nil {
					log.Printf("Wayland dispatch error: %v", err)
					return
				}
			}
		}
	}()

	return nil
}

func (w *WaylandIdleDetector) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.cancel()

	if w.notification != nil {
		w.notification.Destroy()
		w.notification = nil
	}

	if w.display != nil {
		w.display.Context().Close()
	}
}
