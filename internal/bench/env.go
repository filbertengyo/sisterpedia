package bench

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// Environment describes the machine and software used for a benchmark run.
type Environment struct {
	Timestamp  string `json:"timestamp"`
	OS         string `json:"os"`
	OSVersion  string `json:"os_version"`
	Arch       string `json:"arch"`
	CPU        string `json:"cpu"`
	Cores      int    `json:"logical_cores"`
	RAMBytes   uint64 `json:"ram_bytes"`
	GoVersion  string `json:"go_version"`
	Lattigo    string `json:"lattigo"`
	GitCommit  string `json:"git_commit"`
	GitDirty   bool   `json:"git_dirty"`
	GOMAXPROCS int    `json:"gomaxprocs"`
}

func run(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output() // #nosec G204 -- fixed commands
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func lattigoVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, d := range bi.Deps {
			if d.Path == "github.com/tuneinsight/lattigo/v6" {
				return d.Path + " " + d.Version
			}
		}
	}
	return "github.com/tuneinsight/lattigo/v6 v6.2.0"
}

// CaptureEnvironment collects hardware/software information (best effort).
func CaptureEnvironment() Environment {
	e := Environment{
		Timestamp: time.Now().UTC().Format(time.RFC3339), OS: runtime.GOOS, Arch: runtime.GOARCH,
		Cores: runtime.NumCPU(), GoVersion: runtime.Version(), Lattigo: lattigoVersion(), GOMAXPROCS: runtime.GOMAXPROCS(0),
	}
	switch runtime.GOOS {
	case "windows":
		ps := func(cmd string) string { return run("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd) }
		e.CPU = ps("(Get-CimInstance Win32_Processor | Select-Object -First 1).Name")
		if v, err := strconv.ParseUint(ps("(Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory"), 10, 64); err == nil {
			e.RAMBytes = v
		}
		e.OSVersion = ps("$o = Get-CimInstance Win32_OperatingSystem; \"$($o.Caption) $($o.Version)\"")
	case "linux":
		if f, err := os.Open("/proc/cpuinfo"); err == nil {
			s := bufio.NewScanner(f)
			for s.Scan() {
				if strings.HasPrefix(s.Text(), "model name") {
					e.CPU = strings.TrimSpace(strings.SplitN(s.Text(), ":", 2)[1])
					break
				}
			}
			_ = f.Close()
		}
		if f, err := os.Open("/proc/meminfo"); err == nil {
			s := bufio.NewScanner(f)
			for s.Scan() {
				if strings.HasPrefix(s.Text(), "MemTotal:") {
					fields := strings.Fields(s.Text())
					if len(fields) >= 2 {
						if kb, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
							e.RAMBytes = kb * 1024
						}
					}
				}
			}
			_ = f.Close()
		}
		e.OSVersion = run("uname", "-sr")
	case "darwin":
		e.CPU = run("sysctl", "-n", "machdep.cpu.brand_string")
		if v, err := strconv.ParseUint(run("sysctl", "-n", "hw.memsize"), 10, 64); err == nil {
			e.RAMBytes = v
		}
		e.OSVersion = "macOS " + run("sw_vers", "-productVersion")
	}
	e.GitCommit = run("git", "rev-parse", "--short", "HEAD")
	e.GitDirty = run("git", "status", "--porcelain") != ""
	return e
}

// WriteJSON writes v as indented JSON.
func WriteJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
