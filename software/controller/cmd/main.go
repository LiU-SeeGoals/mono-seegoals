package main


import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/trace"
	"syscall"

	"github.com/LiU-SeeGoals/controller/internal/demos"
)

// Use RUN_PROFILER to trace program execution time and memory usage
// When you run this, it will create a file called trace.out. 
// You can then analyze this file using the go tool trace command:
// go tool trace trace.out
// You can also watch the pprof server on localhost:6060 to see live profiling data.
// Todo this, run the command go tool pprof -alloc_space http://localhost:6060/debug/pprof/heap 
// and then write top to see memory usage.

const (
	RUN_PROFILER = false
)

func main() {
	if RUN_PROFILER {
		go func() {
			fmt.Println("Pprof server running on port 6060")
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()

		f, err := os.Create("trace.out")
		if err != nil {
			panic(err)
		}
		defer f.Close()

		err = trace.Start(f)
		if err != nil {
			panic(err)
		}

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
	}

	demos.Defence2BotScenario()
}
