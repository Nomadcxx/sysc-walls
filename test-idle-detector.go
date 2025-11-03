// test-idle-detector.go - Simple tester for Wayland idle detection
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nomadcxx/sysc-walls/pkg/idle"
)

func main() {
	timeout := flag.Int("timeout", 10, "Idle timeout in seconds")
	flag.Parse()

	fmt.Printf("=== Wayland Idle Detection Tester ===\n")
	fmt.Printf("Idle timeout: %d seconds\n", *timeout)
	fmt.Printf("Move mouse or press keys to test resume detection\n")
	fmt.Printf("Press Ctrl+C to exit\n\n")

	idleCount := 0
	resumeCount := 0

	onIdle := func() {
		idleCount++
		fmt.Printf("[%s] 🔴 IDLE detected (count: %d)\n", time.Now().Format("15:04:05"), idleCount)
	}

	onResume := func() {
		resumeCount++
		fmt.Printf("[%s] 🟢 RESUME detected (count: %d)\n", time.Now().Format("15:04:05"), resumeCount)
	}

	fmt.Println("Creating Wayland CGO detector...")
	detector, err := idle.NewWaylandCGODetector(time.Duration(*timeout)*time.Second, onIdle, onResume)
	if err != nil {
		log.Fatalf("Failed to create detector: %v", err)
	}
	defer detector.Stop()

	fmt.Println("Starting detector...")
	if err := detector.Start(); err != nil {
		log.Fatalf("Failed to start detector: %v", err)
	}

	fmt.Printf("\n✓ Detector running! Waiting for idle/resume events...\n\n")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for signal or run indefinitely
	<-sigChan

	fmt.Printf("\n\n=== Test Summary ===\n")
	fmt.Printf("Idle events detected: %d\n", idleCount)
	fmt.Printf("Resume events detected: %d\n", resumeCount)
	fmt.Println("\nShutting down...")
}
