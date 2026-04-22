package main

import (
	"fmt"
	"time"

	"github.com/simulatedsimian/joystick"
)

func main() {
	fmt.Println("Scanning for all input devices...")
	fmt.Println("==========================================")
	
	var controllers []int
	
	// Scan all possible IDs
	for i := 0; i < 10; i++ {
		js, err := joystick.Open(i)
		if err == nil {
			fmt.Printf("\n[ID %d] %s\n", i, js.Name())
			fmt.Printf("  Axes: %d, Buttons: %d\n", js.AxisCount(), js.ButtonCount())
			
			// Xbox controllers typically have 6+ axes and 10+ buttons
			if js.ButtonCount() >= 10 {
				fmt.Printf("  ✓ This looks like a game controller!\n")
				controllers = append(controllers, i)
			} else {
				fmt.Printf("  (Probably not a game controller)\n")
			}
			js.Close()
		}
	}
	
	fmt.Println("\n==========================================")
	
	if len(controllers) == 0 {
		fmt.Println("\n✗ No game controller found!")
		fmt.Println("\nDetected devices don't have enough buttons/axes.")
		fmt.Println("\nTroubleshooting steps:")
		fmt.Println("1. Make sure your Xbox controller is connected (USB or Bluetooth)")
		fmt.Println("2. On Linux, you may need permissions: sudo chmod a+rw /dev/input/js*")
		fmt.Println("3. Check if controller appears: ls -l /dev/input/js*")
		fmt.Println("4. Try: sudo modprobe xpad")
		return
	}
	
	// Use the first real controller found
	controllerID := controllers[0]
	fmt.Printf("\nUsing controller ID %d for testing...\n", controllerID)
	
	js, err := joystick.Open(controllerID)
	if err != nil {
		fmt.Println("Error opening controller:", err)
		return
	}
	defer js.Close()
	
	fmt.Println("\n=== Live Controller Test ===")
	fmt.Println("Move sticks, press buttons, pull triggers...")
	fmt.Println("Press Ctrl+C to exit\n")
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for range ticker.C {
		state, err := js.Read()
		if err != nil {
			fmt.Println("Error reading controller:", err)
			continue
		}
		
		// Display all axis values
		fmt.Print("\rAxes: ")
		for i := 0; i < js.AxisCount(); i++ {
			if i < len(state.AxisData) {
				fmt.Printf("[%d]=%6d ", i, state.AxisData[i])
			}
		}
		
		// Display button states
		fmt.Print("| Buttons: ")
		for i := 0; i < js.ButtonCount() && i < 16; i++ {
			if state.Buttons&(1<<uint(i)) != 0 {
				fmt.Printf("[%d] ", i)
			}
		}
		fmt.Print("    ") // Clear any leftover text
	}
}