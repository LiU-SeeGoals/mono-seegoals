package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/trace"
	"strings"
	"syscall"

	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/controller/internal/runner"
	"github.com/LiU-SeeGoals/controller/internal/scripts"
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
	if _, err := config.Load(); err != nil {
		log.Fatal(err)
	}
	if path := config.EnvFile(); path != "" {
		fmt.Println("Loaded environment from", path)
	}
	if err := logger.Init(config.GetLogDir()); err != nil {
		log.Fatal(err)
	}

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

	runner.Run(scenarioConfig(selectedScenario()))
}

func scenarioConfig(name string) runner.Config {
	sim := config.IsSimulated()
	switch name {
	case "vs-tigers-blue", "vs-tigers-as-blue":
		requireSim(sim)
		return runner.Match(true, info.Blue)
	case "vs-tigers-yellow", "vs-tigers-as-yellow":
		requireSim(sim)
		return runner.Match(true, info.Yellow)
	case "real-blue", "blue":
		return runner.Match(false, info.Blue)
	case "real-yellow", "yellow":
		return runner.Match(false, info.Yellow)
	case "script":
		return runner.Script(sim, info.Blue, scripts.DriveSquare)
	default:
		return runner.Match(sim, info.Yellow, info.Blue)
	}
}

// vs-TIGERs leaves the other team's command port to the TIGERs AI.
func requireSim(sim bool) {
	if !sim {
		panic("vs-TIGERs scenarios require ENVIRONMENT=simulation")
	}
}

func selectedScenario() string {
	if len(os.Args) > 1 {
		return strings.ToLower(os.Args[1])
	}
	return strings.ToLower(os.Getenv("SEEGOALS_SCENARIO"))
}
