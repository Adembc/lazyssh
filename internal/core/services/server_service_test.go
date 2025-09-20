package services

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/Adembc/lazyssh/internal/core/ports"
	"go.uber.org/zap/zaptest"
)

type mockServerRepository struct {
	ports.ServerRepository
	recordCalls int
	lastAlias   string
	recordErr   error
}

func (m *mockServerRepository) ListServers(string) ([]domain.Server, error) {
	return nil, nil
}

func (m *mockServerRepository) UpdateServer(domain.Server, domain.Server) error { return nil }

func (m *mockServerRepository) AddServer(domain.Server) error { return nil }

func (m *mockServerRepository) DeleteServer(domain.Server) error { return nil }

func (m *mockServerRepository) SetPinned(string, bool) error { return nil }

func (m *mockServerRepository) RecordSSH(alias string) error {
	m.recordCalls++
	m.lastAlias = alias
	return m.recordErr
}

func helperCommandFactory(scenario string) func(string) *exec.Cmd {
	return func(alias string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", scenario, alias)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		return cmd
	}
}

func TestServerServiceSSH_RemoteDisconnect(t *testing.T) {
	repo := &mockServerRepository{}
	svc := &serverService{
		logger:           zaptest.NewLogger(t).Sugar(),
		serverRepository: repo,
		newSSHCommand:    helperCommandFactory("remote"),
	}

	if err := svc.SSH("example"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if repo.recordCalls != 1 {
		t.Fatalf("expected RecordSSH to be called once, got %d", repo.recordCalls)
	}
}

func TestServerServiceSSH_ConnectionReset(t *testing.T) {
	repo := &mockServerRepository{}
	svc := &serverService{
		logger:           zaptest.NewLogger(t).Sugar(),
		serverRepository: repo,
		newSSHCommand:    helperCommandFactory("reset"),
	}

	if err := svc.SSH("example"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if repo.recordCalls != 1 {
		t.Fatalf("expected RecordSSH to be called once, got %d", repo.recordCalls)
	}
}

func TestServerServiceSSH_PermissionDenied(t *testing.T) {
	repo := &mockServerRepository{}
	svc := &serverService{
		logger:           zaptest.NewLogger(t).Sugar(),
		serverRepository: repo,
		newSSHCommand:    helperCommandFactory("permission"),
	}

	if err := svc.SSH("example"); err == nil {
		t.Fatalf("expected error, got nil")
	}

	if repo.recordCalls != 0 {
		t.Fatalf("expected RecordSSH not to be called, got %d", repo.recordCalls)
	}
}

func TestServerServiceSSH_Success(t *testing.T) {
	repo := &mockServerRepository{}
	svc := &serverService{
		logger:           zaptest.NewLogger(t).Sugar(),
		serverRepository: repo,
		newSSHCommand:    helperCommandFactory("success"),
	}

	if err := svc.SSH("example"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if repo.recordCalls != 1 {
		t.Fatalf("expected RecordSSH to be called once, got %d", repo.recordCalls)
	}
}

func TestServerServiceSSH_CommandFactoryReturnsNil(t *testing.T) {
	repo := &mockServerRepository{}
	svc := &serverService{
		logger:           zaptest.NewLogger(t).Sugar(),
		serverRepository: repo,
		newSSHCommand: func(string) *exec.Cmd {
			return nil
		},
	}

	err := svc.SSH("example")
	if err == nil {
		t.Fatalf("expected error when command factory returns nil")
	}
}

func TestIsRemoteDisconnectError(t *testing.T) {
	if isRemoteDisconnectError(errors.New("plain error"), "Connection closed by remote host") {
		t.Fatalf("expected non-exit error to return false")
	}

	cmd := helperCommandFactory("remote")("example")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error for remote disconnect scenario")
	}
	if !isRemoteDisconnectError(err, "Connection to example closed by remote host.\n") {
		t.Fatalf("expected remote disconnect error to be detected")
	}

	cmd = helperCommandFactory("permission")("example")
	err = cmd.Run()
	if err == nil {
		t.Fatalf("expected error for permission scenario")
	}
	if isRemoteDisconnectError(err, "Permission denied (publickey).\n") {
		t.Fatalf("did not expect permission error to be treated as disconnect")
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for i := 0; i < len(args); i++ {
		if args[i] == "--" && i+1 < len(args) {
			scenario := args[i+1]
			alias := ""
			if i+2 < len(args) {
				alias = args[i+2]
			}

			switch scenario {
			case "remote":
				_, _ = os.Stderr.WriteString("Connection to " + alias + " closed by remote host.\n")
				os.Exit(255)
			case "reset":
				_, _ = os.Stderr.WriteString("Read from remote host " + alias + ": Connection reset by peer\n")
				os.Exit(255)
			case "permission":
				_, _ = os.Stderr.WriteString("Permission denied (publickey).\n")
				os.Exit(255)
			case "success":
				os.Exit(0)
			default:
				_, _ = os.Stderr.WriteString("unknown scenario\n")
				os.Exit(1)
			}
		}
	}
	os.Exit(1)
}
