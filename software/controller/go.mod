module github.com/LiU-SeeGoals/controller

go 1.21.2

toolchain go1.22.0

require (
	// github.com/golang/protobuf v1.5.0
	gonum.org/v1/gonum v0.14.0
	google.golang.org/protobuf v1.32.0
)

require (
	github.com/LiU-SeeGoals/proto_go v0.1.3-0.20241001130039-d2d8519ef393
	github.com/caarlos0/env/v10 v10.0.0
	github.com/eiannone/keyboard v0.0.0-20220611211555-0d226195f203
	github.com/gorilla/websocket v1.5.1
	// google.golang.org/protobuf v1.31.0
	github.com/joho/godotenv v1.5.1

// golang.org/x/net v1.2.3
)

require golang.org/x/sys v0.20.0 // indirect

// Generated from software/proto-messages; regenerate with its build_go.sh.
replace github.com/LiU-SeeGoals/proto_go => ../proto-messages/proto_go

require (
	go.uber.org/zap v1.27.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require (
	github.com/veandco/go-sdl2 v0.4.40 // used by internal/ui/controller_remote_control.go (go run)
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/text v0.16.0 // indirect
)
