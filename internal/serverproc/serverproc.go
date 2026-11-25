// Package serverproc builds the real cmd/server binary and runs it as a
// separate OS process with a temporary database, for end-to-end tests and
// benchmarks. It supports stopping and restarting the same database to test
// persistence.
package serverproc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Server is a running (or stopped) server process.
type Server struct {
	Bin     string
	DBPath  string
	Addr    string // host:port
	URL     string
	Env     []string
	RepoDir string

	mu  sync.Mutex
	cmd *exec.Cmd
	out *syncBuffer
}

// syncBuffer is a goroutine-safe output buffer (exec copies output concurrently).
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// RepoRoot walks up from the working directory to the directory holding go.mod.
func RepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found")
		}
		dir = parent
	}
}

// Build compiles cmd/server into dir and returns the binary path.
func Build(repo, dir string) (string, error) {
	bin := filepath.Join(dir, "server")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/server") // #nosec G204 -- fixed arguments
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("build server: %w\n%s", err, out)
	}
	return bin, nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// New prepares a server process on a free port with a database in dataDir.
func New(repo, bin, dataDir string, extraEnv ...string) (*Server, error) {
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	s := &Server{
		Bin: bin, DBPath: filepath.Join(dataDir, "shop.db"), Addr: addr, URL: "http://" + addr, RepoDir: repo,
	}
	s.Env = append(os.Environ(),
		"SERVER_ADDR="+addr,
		"DB_PATH="+s.DBPath,
		"CATALOG_PATH="+filepath.Join(repo, "data", "catalog.json"),
		"CKKS_PROFILES_FILE="+filepath.Join(repo, "config", "profiles.yaml"),
		"ALLOWED_PROFILES=p1,p2",
		"ALLOWED_ORIGIN=http://localhost:3000",
	)
	s.Env = append(s.Env, extraEnv...)
	return s, nil
}

// Start launches the process and waits for /healthz.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != nil {
		return errors.New("already running")
	}
	cmd := exec.Command(s.Bin) // #nosec G204 -- binary built by Build
	cmd.Env = s.Env
	cmd.Dir = s.RepoDir
	if s.out == nil {
		s.out = &syncBuffer{} // kept across restarts
	}
	cmd.Stdout = s.out
	cmd.Stderr = s.out
	if err := cmd.Start(); err != nil {
		return err
	}
	s.cmd = cmd
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.URL+"/healthz", nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("server did not become healthy: %s", s.out.String())
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// Stop kills the process (committed SQLite transactions are durable).
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd == nil {
		return nil
	}
	_ = s.cmd.Process.Kill()
	_ = s.cmd.Wait()
	s.cmd = nil
	return nil
}

// Restart stops and starts the process on the same database and port.
func (s *Server) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}
	return s.Start()
}

// PID returns the current process id (0 if stopped).
func (s *Server) PID() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd == nil || s.cmd.Process == nil {
		return 0
	}
	return s.cmd.Process.Pid
}

// Logs returns the server's combined output (access log; never bodies).
func (s *Server) Logs() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.out == nil {
		return ""
	}
	return s.out.String()
}
