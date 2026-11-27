// Package config reads the controller's settings from the process environment.
//
// Every setting has a default matching the repository's environment.ini, so
// tests and local runs work without any .env file. A .env file is optional:
// Load reads the file named by SEEGOALS_ENV_FILE, or else the first .env found
// in the working directory or one of its parents. Variables already present in
// the process environment always win over the file.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

// EnvFileVar names an explicit .env file to load instead of searching for one.
const EnvFileVar = "SEEGOALS_ENV_FILE"

type Config struct {
	// "simulation" selects simulator addresses and outputs; anything else is
	// treated as real hardware.
	Env string `env:"ENVIRONMENT" envDefault:"simulation"`

	SSLVision        ConfigSSLVision
	SSLTrackedVision ConfigSSLTrackedVision
	Real             ConfigReal
	Sim              ConfigSim
	GC               ConfigGameController
	GW               ConfigGameViewer
	BS               ConfigBasestation
	AI               ConfigAI
	Log              ConfigLog
}

type ConfigSSLVision struct {
	Address   string `env:"SSL_VISION_MULTICAST_ADDR" envDefault:"224.5.23.2"`
	Port      string `env:"SSL_VISION_SIM_MAIN_PORT" envDefault:"10020"`
	Port_real string `env:"SSL_VISION_REAL_MAIN_PORT" envDefault:"10006"`
	VizPort   string `env:"SSL_VISION_VIZ_PORT" envDefault:"11011"`
}

type ConfigSSLTrackedVision struct {
	Address string `env:"SSL_VISION_MULTICAST_ADDR" envDefault:"224.5.23.2"`
	Port    string `env:"SSL_TRACKER_PORT" envDefault:"10010"`
}

// ConfigReal holds the basestation's own unicast address. Commands are sent to
// the multicast group in ConfigBasestation, which the basestation joins.
type ConfigReal struct {
	BASESTATION_ADDR string `env:"BASESTATION_ADDR" envDefault:"192.168.1.150"`
	BASESTATION_PORT string `env:"BASESTATION_PORT" envDefault:"6001"`
}

type ConfigGameViewer struct {
	Address string `env:"VITE_AI_GAME_VIEWER_SOCKET_ADDR" envDefault:"127.0.0.1"`
	Port    string `env:"VITE_AI_GAME_VIEWER_SOCKET_PORT" envDefault:"5174"`
}

type ConfigBasestation struct {
	Address string `env:"AI_ACTIONS_BASESTATION_MULTICAST_ADDR" envDefault:"239.0.0.2"`
	Port    string `env:"AI_ACTIONS_BASESTATION_MULTICAST_PORT" envDefault:"9999"`
}

type ConfigAI struct {
	// Empty means every multicast-capable interface.
	Interface                  string `env:"AI_MULTICAST_INTERFACE" envDefault:""`
	RestrictToOpponentHalfTeam string `env:"AI_RESTRICT_ROBOTS_TO_OPPONENT_HALF" envDefault:""`
}

type ConfigSim struct {
	Address              string `env:"SIM_ADDR" envDefault:"127.0.0.1"`
	CommandPort          string `env:"SIM_COMMAND_LISTEN_PORT" envDefault:"20011"`
	BlueStatusPort       string `env:"SIM_BLUE_STATUS_SEND_PORT" envDefault:"30011"`
	YellowStatusPort     string `env:"SIM_YELLOW_STATUS_SEND_PORT" envDefault:"30012"`
	SimControllerPort    string `env:"SIM_SIM_CONTROLLER_SEND_PORT" envDefault:"10300"`
	BlueControllerPort   string `env:"SIM_BLUE_CONTROLLER_LISTEN_PORT" envDefault:"10301"`
	YellowControllerPort string `env:"SIM_YELLOW_CONTROLLER_LISTEN_PORT" envDefault:"10302"`

	// Limits of the simulator-side motion controller (client.SimMotion).
	// The defaults reproduce the controller's historical behaviour; the real
	// robots run their own onboard controller and document no limits.
	MaxLinearMPS    float64 `env:"SIM_MAX_LINEAR_MPS" envDefault:"1.0"`
	MaxAngularRadPS float64 `env:"SIM_MAX_ANGULAR_RADPS" envDefault:"3.0"`
	// Zero disables acceleration limiting.
	MaxLinearAccMPS2 float64 `env:"SIM_MAX_LINEAR_ACC_MPS2" envDefault:"0"`
}

type ConfigGameController struct {
	Address string `env:"SSL_GAME_CONTROLLER_PUBLISH_ADDR" envDefault:"224.5.23.1"`
	Port    string `env:"SSL_GAME_CONTROLLER_PUBLISH_PORT" envDefault:"10003"`
}

type ConfigLog struct {
	// Directory for readable.log and structured.log, relative to the working
	// directory. Empty disables file logging.
	Dir string `env:"SEEGOALS_LOG_DIR" envDefault:"../logs"`
}

var (
	mu       sync.Mutex
	instance *Config
	envFile  string
)

// Load reads the optional .env file and parses the environment. It is safe to
// call more than once; the first successful result is kept.
func Load() (*Config, error) {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance, nil
	}

	path, err := findEnvFile()
	if err != nil {
		return nil, err
	}
	if path != "" {
		// godotenv.Load never overrides variables already in the environment.
		if err := godotenv.Load(path); err != nil {
			return nil, fmt.Errorf("config: load %s: %w", path, err)
		}
	}

	cfg, err := Parse()
	if err != nil {
		return nil, err
	}
	instance, envFile = cfg, path
	return cfg, nil
}

// Parse builds a Config from the current process environment only.
func Parse() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

// EnvFile reports which .env file Load used ("" when none).
func EnvFile() string {
	mu.Lock()
	defer mu.Unlock()
	return envFile
}

func findEnvFile() (string, error) {
	if explicit := os.Getenv(EnvFileVar); explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("config: %s=%s: %w", EnvFileVar, explicit, err)
		}
		return explicit, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", nil
	}
	for {
		candidate := filepath.Join(dir, ".env")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("config: %w", err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// GetInstance returns the loaded config, loading it on first use. A malformed
// value (e.g. a non-numeric speed limit) is a programming/deployment error and
// panics with the parse error.
func GetInstance() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

func join(host, port string) string { return host + ":" + port }

func GetGCClientAddress() string {
	cfg := GetInstance()
	return join(cfg.GC.Address, cfg.GC.Port)
}

func GetSSLClientAddress() string {
	cfg := GetInstance()
	return join(cfg.SSLVision.Address, cfg.SSLVision.Port)
}

func GetSSLClientAddressReal() string {
	cfg := GetInstance()
	return join(cfg.SSLVision.Address, cfg.SSLVision.Port_real)
}

func GetSSLTrackedClientAddress() string {
	cfg := GetInstance()
	return join(cfg.SSLTrackedVision.Address, cfg.SSLTrackedVision.Port)
}

func GetSimControlAddress() string {
	cfg := GetInstance()
	return join(cfg.Sim.Address, cfg.Sim.SimControllerPort)
}

func GetSimBlueTeamAddress() string {
	cfg := GetInstance()
	return join(cfg.Sim.Address, cfg.Sim.BlueControllerPort)
}

func GetSimYellowTeamAddress() string {
	cfg := GetInstance()
	return join(cfg.Sim.Address, cfg.Sim.YellowControllerPort)
}

// GetBasestationMulticastAddress is where robot commands are sent; the
// basestation firmware joins this group.
func GetBasestationMulticastAddress() string {
	cfg := GetInstance()
	return join(cfg.BS.Address, cfg.BS.Port)
}

func GetAIMulticastInterface() string {
	return GetInstance().AI.Interface
}

func RestrictToOpponentHalfTeam() string {
	return GetInstance().AI.RestrictToOpponentHalfTeam
}

func GetGameViewerPort() string {
	return ":" + GetInstance().GW.Port
}

func GetLogDir() string {
	return GetInstance().Log.Dir
}

func IsSimulated() bool {
	return GetInstance().Env == "simulation"
}
