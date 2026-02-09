package main

import (
	"fmt"
	"log"
	"net/http"          // <--- CHANGE 1: Import net/http
	_ "net/http/pprof"  // <--- CHANGE 2: Import pprof for side effects
	"os"
	"os/signal"
	"runtime/trace"
	"syscall"

	"github.com/LiU-SeeGoals/controller/internal/demos"
)

func main() {
	// --- CHANGE 3: Start the PPROF server in the background ---
	go func() {
		fmt.Println("Pprof server running on port 6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	// ----------------------------------------------------------

	// 1. Create the trace file
	f, err := os.Create("trace.out")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// 2. Start the trace
	err = trace.Start(f)
	if err != nil {
		panic(err)
	}

	// 3. Setup signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		fmt.Println("\nReceived interrupt signal, stopping trace...")
		trace.Stop()
		f.Close()
		os.Exit(0)
	}()

	defer trace.Stop()

	fmt.Println("Running scenario... Press Ctrl+C to stop.")
	demos.FwRealScenario()
}