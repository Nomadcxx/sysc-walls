// test-wayland-raw.go - Minimal Wayland test with just CGO, no daemon infrastructure
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nomadcxx/sysc-walls/pkg/idle"
)

func main() {
	fmt.Println("=== Minimal Wayland CGO Test ===")
	fmt.Println("This test uses the EXACT same code as the daemon")
	fmt.Println("Timeout: 10 seconds")
	fmt.Println("")

	idleCount := 0
	resumeCount := 0
	lastEvent := time.Now()

	onIdle := func() {
		idleCount++
		now := time.Now()
		fmt.Printf("[%s] 🔴 IDLE #%d (%.1fs since last event)\n", 
			now.Format("15:04:05"), idleCount, now.Sub(lastEvent).Seconds())
		lastEvent = now
	}

	onResume := func() {
		resumeCount++
		now := time.Now()
		fmt.Printf("[%s] 🟢 RESUME #%d (%.1fs since last event)\n", 
			now.Format("15:04:05"), resumeCount, now.Sub(lastEvent).Seconds())
		lastEvent = now
	}

	detector, err := idle.NewWaylandCGODetector(10*time.Second, onIdle, onResume)
	if err != nil {
		log.Fatalf("❌ Failed to create detector: %v", err)
	}
	defer detector.Stop()

	if err := detector.Start(); err != nil {
		log.Fatalf("❌ Failed to start detector: %v", err)
	}

	fmt.Println("✓ Detector running")
	fmt.Println("")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Idle events:   %d\n", idleCount)
	fmt.Printf("Resume events: %d\n", resumeCount)
}
