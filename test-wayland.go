package main

import (
	
	"log"
	"time"

	"github.com/rajveermalviya/go-wayland/wayland/client"
	ext_idle_notify "github.com/rajveermalviya/go-wayland/wayland/staging/ext-idle-notify-v1"
)

func main() {
	log.Println("Connecting to Wayland...")
	display, err := client.Connect("")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer display.Context().Close()

	log.Println("Getting registry...")
	registry, err := display.GetRegistry()
	if err != nil {
		log.Fatalf("Failed to get registry: %v", err)
	}

	var idleNotifierName, idleNotifierVersion uint32
	var seatName, seatVersion uint32

	registry.SetGlobalHandler(func(e client.RegistryGlobalEvent) {
		log.Printf("Global: %s (name=%d, version=%d)", e.Interface, e.Name, e.Version)
		switch e.Interface {
		case "ext_idle_notifier_v1":
			idleNotifierName = e.Name
			idleNotifierVersion = e.Version
		case "wl_seat":
			if seatName == 0 {
				seatName = e.Name
				seatVersion = e.Version
			}
		}
	})

	log.Println("Roundtrip 1...")
	roundtrip(display)
	log.Println("Roundtrip 2...")
	roundtrip(display)

	if idleNotifierName == 0 {
		log.Fatalf("ext_idle_notifier_v1 not found")
	}
	if seatName == 0 {
		log.Fatalf("wl_seat not found")
	}

	log.Printf("Found ext_idle_notifier_v1 (name=%d) and wl_seat (name=%d)", idleNotifierName, seatName)

	log.Println("Binding to idle notifier...")
	idleNotifier := ext_idle_notify.NewIdleNotifier(display.Context())
	if err := registry.Bind(idleNotifierName, "ext_idle_notifier_v1", idleNotifierVersion, idleNotifier); err != nil {
		log.Fatalf("Failed to bind: %v", err)
	}

	log.Println("Binding to seat...")
	seat := client.NewSeat(display.Context())
	if err := registry.Bind(seatName, "wl_seat", seatVersion, seat); err != nil {
		log.Fatalf("Failed to bind seat: %v", err)
	}

	log.Println("Creating idle notification...")
	notification, err := idleNotifier.GetIdleNotification(5000, seat) // 5 seconds
	if err != nil {
		log.Fatalf("Failed to get notification: %v", err)
	}

	notification.SetIdledHandler(func(e ext_idle_notify.IdleNotificationIdledEvent) {
		log.Println("IDLE!")
	})

	notification.SetResumedHandler(func(e ext_idle_notify.IdleNotificationResumedEvent) {
		log.Println("RESUMED!")
	})

	log.Println("Starting event loop...")
	for i := 0; i < 60; i++ {
		if err := display.Context().Dispatch(); err != nil {
			log.Printf("Dispatch error: %v", err)
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	log.Println("Done")
}

func roundtrip(display *client.Display) {
	callback, err := display.Sync()
	if err != nil {
		log.Fatalf("Failed to sync: %v", err)
	}
	defer callback.Destroy()

	done := false
	callback.SetDoneHandler(func(_ client.CallbackDoneEvent) {
		done = true
	})

	for !done {
		if err := display.Context().Dispatch(); err != nil {
			log.Fatalf("Roundtrip dispatch error: %v", err)
		}
	}
}
