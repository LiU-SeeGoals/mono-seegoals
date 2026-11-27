package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDefaultsWithoutEnvironment(t *testing.T) {
	t.Setenv("ENVIRONMENT", "")
	os.Unsetenv("ENVIRONMENT")
	cfg, err := Parse()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "simulation" {
		t.Fatalf("Env = %q, want simulation", cfg.Env)
	}
	if got := join(cfg.BS.Address, cfg.BS.Port); got != "239.0.0.2:9999" {
		t.Fatalf("basestation multicast = %s", got)
	}
	if cfg.Sim.MaxLinearMPS != 1.0 || cfg.Sim.MaxAngularRadPS != 3.0 || cfg.Sim.MaxLinearAccMPS2 != 0 {
		t.Fatalf("unexpected sim limits %+v", cfg.Sim)
	}
}

func TestParseEnvironmentOverridesDefault(t *testing.T) {
	t.Setenv("SSL_TRACKER_PORT", "12345")
	t.Setenv("SIM_MAX_LINEAR_MPS", "2.5")
	cfg, err := Parse()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SSLTrackedVision.Port != "12345" || cfg.Sim.MaxLinearMPS != 2.5 {
		t.Fatalf("overrides not applied: %+v %+v", cfg.SSLTrackedVision, cfg.Sim)
	}
}

func TestParseRejectsMalformedNumber(t *testing.T) {
	t.Setenv("SIM_MAX_LINEAR_MPS", "fast")
	if _, err := Parse(); err == nil {
		t.Fatal("expected an error for a non-numeric speed limit")
	}
}

func TestFindEnvFileSearchesParents(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(root, ".env")
	if err := os.WriteFile(envPath, []byte("ENVIRONMENT=real\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvFileVar, "")
	wd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(wd) })
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	got, err := findEnvFile()
	if err != nil {
		t.Fatal(err)
	}
	if got != envPath {
		t.Fatalf("findEnvFile = %q, want %q", got, envPath)
	}
}

func TestFindEnvFileExplicitMissingIsError(t *testing.T) {
	t.Setenv(EnvFileVar, filepath.Join(t.TempDir(), "missing.env"))
	if _, err := findEnvFile(); err == nil {
		t.Fatal("expected an error for a missing explicit env file")
	}
}
