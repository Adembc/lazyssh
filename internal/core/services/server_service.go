// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/Adembc/lazyssh/internal/core/ports"
	"go.uber.org/zap"
)

type serverService struct {
	serverRepository ports.ServerRepository
	logger           *zap.SugaredLogger

	fwMu     sync.Mutex
	forwards map[string][]*os.Process
}

type activeSSHSession struct {
	alias          string
	host           string
	user           string
	port           int
	identityFiles  []string
	localForward   []string
	remoteForward  []string
	dynamicForward []string
	pid            int
}

// NewServerService creates a new instance of serverService.
func NewServerService(logger *zap.SugaredLogger, sr ports.ServerRepository) ports.ServerService {
	return &serverService{
		logger:           logger,
		serverRepository: sr,
	}
}

// ListServers returns a list of servers sorted with pinned on top.
func (s *serverService) ListServers(query string) ([]domain.Server, error) {
	servers, err := s.serverRepository.ListServers(query)
	if err != nil {
		s.logger.Errorw("failed to list servers", "error", err)
		return nil, err
	}

	// Sort: pinned first (PinnedAt non-zero), then by PinnedAt desc, then by Alias asc.
	sort.SliceStable(servers, func(i, j int) bool {
		pi := !servers[i].PinnedAt.IsZero()
		pj := !servers[j].PinnedAt.IsZero()
		if pi != pj {
			return pi
		}
		if pi && pj {
			return servers[i].PinnedAt.After(servers[j].PinnedAt)
		}
		return servers[i].Alias < servers[j].Alias
	})

	return servers, nil
}

// ListActiveSessions returns currently running SSH sessions as list entries.
func (s *serverService) ListActiveSessions(query string) ([]domain.Server, error) {
	activeSessions, err := s.listActiveSSHSessions()
	if err != nil {
		return nil, err
	}

	configured, err := s.serverRepository.ListServers("")
	if err != nil {
		return nil, err
	}

	aliasIndex := make(map[string]domain.Server, len(configured))
	hostIndex := make(map[string]domain.Server, len(configured))
	for _, server := range configured {
		aliasIndex[strings.ToLower(server.Alias)] = server
		for _, alias := range server.Aliases {
			aliasIndex[strings.ToLower(alias)] = server
		}
		if server.Host != "" {
			hostIndex[strings.ToLower(server.Host)] = server
		}
	}

	query = strings.ToLower(strings.TrimSpace(query))
	entries := make([]domain.Server, 0, len(activeSessions))
	for _, session := range activeSessions {
		entry := domain.Server{}
		if session.alias != "" {
			if server, ok := aliasIndex[strings.ToLower(session.alias)]; ok {
				entry = server
			}
		}
		if entry.Alias == "" && session.host != "" {
			if server, ok := hostIndex[strings.ToLower(session.host)]; ok {
				entry = server
			}
		}

		if entry.Alias == "" {
			if session.alias != "" {
				entry.Alias = session.alias
			} else {
				entry.Alias = "unknown"
			}
			entry.Aliases = []string{entry.Alias}
		}

		if session.host != "" {
			entry.Host = session.host
		} else if entry.Host == "" {
			entry.Host = "unknown"
		}
		if session.user != "" {
			entry.User = session.user
		}
		if session.port > 0 {
			entry.Port = session.port
		} else if entry.Port == 0 {
			entry.Port = 22
		}

		entry.IdentityFiles = mergeIdentityFiles(entry.IdentityFiles, session.identityFiles)
		entry.LocalForward = mergeForwardSpecs(entry.LocalForward, session.localForward)
		entry.RemoteForward = mergeForwardSpecs(entry.RemoteForward, session.remoteForward)
		entry.DynamicForward = mergeForwardSpecs(entry.DynamicForward, session.dynamicForward)
		entry.ActivePID = session.pid
		entry.LastSeen = time.Now()
		entry.Tags = append([]string{"active"}, entry.Tags...)

		if query != "" && !matchesServerQuery(entry, query) {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// validateServer performs core validation of server fields.
func validateServer(srv domain.Server) error {
	if strings.TrimSpace(srv.Alias) == "" {
		return fmt.Errorf("alias is required")
	}
	if ok, _ := regexp.MatchString(`^[A-Za-z0-9_.-]+$`, srv.Alias); !ok {
		return fmt.Errorf("alias may contain letters, digits, dot, dash, underscore")
	}
	if strings.TrimSpace(srv.Host) == "" {
		return fmt.Errorf("Host/IP is required")
	}
	if ip := net.ParseIP(srv.Host); ip == nil {
		if strings.Contains(srv.Host, " ") {
			return fmt.Errorf("host must not contain spaces")
		}
		if ok, _ := regexp.MatchString(`^[A-Za-z0-9.-]+$`, srv.Host); !ok {
			return fmt.Errorf("host contains invalid characters")
		}
		if strings.HasPrefix(srv.Host, ".") || strings.HasSuffix(srv.Host, ".") {
			return fmt.Errorf("host must not start or end with a dot")
		}
		for _, lbl := range strings.Split(srv.Host, ".") {
			if lbl == "" {
				return fmt.Errorf("host must not contain empty labels")
			}
			if strings.HasPrefix(lbl, "-") || strings.HasSuffix(lbl, "-") {
				return fmt.Errorf("hostname labels must not start or end with a hyphen")
			}
		}
	}
	if srv.Port != 0 && (srv.Port < 1 || srv.Port > 65535) {
		return fmt.Errorf("port must be a number between 1 and 65535")
	}
	return nil
}

// UpdateServer updates an existing server with new details.
func (s *serverService) UpdateServer(server domain.Server, newServer domain.Server) error {
	if err := validateServer(newServer); err != nil {
		s.logger.Warnw("validation failed on update", "error", err, "server", newServer)
		return err
	}
	err := s.serverRepository.UpdateServer(server, newServer)
	if err != nil {
		s.logger.Errorw("failed to update server", "error", err, "server", server)
	}
	return err
}

// AddServer adds a new server to the repository.
func (s *serverService) AddServer(server domain.Server) error {
	if err := validateServer(server); err != nil {
		s.logger.Warnw("validation failed on add", "error", err, "server", server)
		return err
	}
	err := s.serverRepository.AddServer(server)
	if err != nil {
		s.logger.Errorw("failed to add server", "error", err, "server", server)
	}
	return err
}

// DeleteServer removes a server from the repository.
func (s *serverService) DeleteServer(server domain.Server) error {
	err := s.serverRepository.DeleteServer(server)
	if err != nil {
		s.logger.Errorw("failed to delete server", "error", err, "server", server)
	}
	return err
}

// SetPinned sets or clears a pin timestamp for the server alias.
func (s *serverService) SetPinned(alias string, pinned bool) error {
	err := s.serverRepository.SetPinned(alias, pinned)
	if err != nil {
		s.logger.Errorw("failed to set pin state", "error", err, "alias", alias, "pinned", pinned)
	}
	return err
}

// SSH starts an interactive SSH session to the given alias using the system's ssh client.
func (s *serverService) SSH(alias string) error {
	s.logger.Infow("ssh start", "alias", alias)
	cmd := exec.Command("ssh", alias)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		s.logger.Errorw("ssh command failed", "alias", alias, "error", err)
		return err
	}

	if err := s.serverRepository.RecordSSH(alias); err != nil {
		s.logger.Errorw("failed to record ssh metadata", "alias", alias, "error", err)
	}

	s.logger.Infow("ssh end", "alias", alias)
	return nil
}

// SSHWithArgs runs system ssh with provided extra args (e.g., -L/-R/-D) for the given alias.
func (s *serverService) SSHWithArgs(alias string, extraArgs []string) error {
	s.logger.Infow("ssh start (with args)", "alias", alias, "args", extraArgs)
	args := append([]string{}, extraArgs...)
	args = append(args, alias)
	// #nosec G204
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		s.logger.Errorw("ssh (with args) failed", "alias", alias, "error", err)
		return err
	}
	if err := s.serverRepository.RecordSSH(alias); err != nil {
		s.logger.Errorw("failed to record ssh metadata", "alias", alias, "error", err)
	}
	s.logger.Infow("ssh end (with args)", "alias", alias)
	return nil
}

// StartForward starts ssh port forwarding in the background and tracks the process.
func (s *serverService) StartForward(alias string, extraArgs []string) (int, error) {
	s.fwMu.Lock()
	if s.forwards == nil {
		s.forwards = make(map[string][]*os.Process)
	}
	s.fwMu.Unlock()

	extraArgs = append(extraArgs, "-N", alias)

	// #nosec G204
	cmd := exec.Command("ssh", extraArgs...)

	// Detach from TTY: discard stdio
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to open devnull: %w", err)
	}
	defer func() {
		if devNull != nil {
			_ = devNull.Close()
		}
	}()

	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	// Set SysProcAttr in an OS-specific way (see sysprocattr_* files)
	sysProcAttr := &syscall.SysProcAttr{}
	setDetach(sysProcAttr)
	cmd.SysProcAttr = sysProcAttr

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start ssh: %w", err)
	}

	proc := cmd.Process
	if proc == nil {
		return 0, fmt.Errorf("process is nil after start")
	}
	pid := proc.Pid

	// Track process
	s.fwMu.Lock()
	s.forwards[alias] = append(s.forwards[alias], proc)
	s.fwMu.Unlock()

	// Cleanup on exit
	go func(a string, c *exec.Cmd, dn *os.File) {
		_ = c.Wait()
		_ = dn.Close()

		s.fwMu.Lock()
		defer s.fwMu.Unlock()

		procs := s.forwards[a]
		if len(procs) == 0 {
			return
		}

		filtered := make([]*os.Process, 0, len(procs))
		for _, p := range procs {
			if p != nil && p.Pid != pid {
				filtered = append(filtered, p)
			}
		}

		if len(filtered) == 0 {
			delete(s.forwards, a)
		} else {
			s.forwards[a] = filtered
		}
	}(alias, cmd, devNull)

	devNull = nil // Prevent defer from closing it

	return pid, nil
}

// StopForwarding kills all active forward processes for the alias.
func (s *serverService) StopForwarding(alias string) error {
	s.fwMu.Lock()
	procs := s.forwards[alias]
	delete(s.forwards, alias)
	s.fwMu.Unlock()

	if len(procs) == 0 {
		return nil
	}

	var errs []error
	for _, p := range procs {
		if p != nil {
			if err := p.Signal(syscall.SIGTERM); err != nil {
				// If SIGTERM fails, try SIGKILL
				if killErr := p.Kill(); killErr != nil {
					errs = append(errs, fmt.Errorf("failed to kill pid %d: %w", p.Pid, killErr))
				}
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping forwards: %v", errs)
	}
	return nil
}

// IsForwarding reports whether there is at least one active forward for alias.
func (s *serverService) IsForwarding(alias string) bool {
	s.fwMu.Lock()
	defer s.fwMu.Unlock()
	return len(s.forwards[alias]) > 0
}

// KillActiveSessions terminates active SSH sessions matching the server.
func (s *serverService) KillActiveSessions(server domain.Server) (int, error) {
	sessions, err := s.listActiveSSHSessions()
	if err != nil {
		return 0, err
	}

	if server.ActivePID > 0 {
		for _, session := range sessions {
			if session.pid == server.ActivePID {
				if err := killPID(session.pid); err != nil {
					return 0, err
				}
				return 1, nil
			}
		}
		return 0, fmt.Errorf("active ssh session not found")
	}

	var pids []int
	for _, session := range sessions {
		if matchSessionForServer(server, session) && session.pid > 0 {
			pids = append(pids, session.pid)
		}
	}
	if len(pids) == 0 {
		return 0, fmt.Errorf("no active ssh sessions found")
	}

	var errs []error
	killed := 0
	for _, pid := range pids {
		if killErr := killPID(pid); killErr != nil {
			errs = append(errs, fmt.Errorf("pid %d: %w", pid, killErr))
			continue
		}
		killed++
	}

	if len(errs) > 0 {
		return killed, fmt.Errorf("failed to terminate sessions: %v", errs)
	}
	return killed, nil
}

// ResolveConfigServer attempts to map a server entry to a configured server.
func (s *serverService) ResolveConfigServer(server domain.Server) (domain.Server, bool, error) {
	servers, err := s.serverRepository.ListServers("")
	if err != nil {
		return domain.Server{}, false, err
	}
	for _, candidate := range servers {
		if strings.EqualFold(candidate.Alias, server.Alias) {
			return candidate, true, nil
		}
		for _, alias := range candidate.Aliases {
			if strings.EqualFold(alias, server.Alias) {
				return candidate, true, nil
			}
		}
		if server.Host != "" && candidate.Host != "" {
			if strings.EqualFold(candidate.Host, server.Host) {
				return candidate, true, nil
			}
		}
	}
	return domain.Server{}, false, nil
}

// Ping checks if the server is reachable on its SSH port.
func (s *serverService) Ping(server domain.Server) (bool, time.Duration, error) {
	start := time.Now()

	host, port, ok := resolveSSHDestination(server.Alias)
	if !ok {

		host = strings.TrimSpace(server.Host)
		if host == "" {
			host = server.Alias
		}
		if server.Port > 0 {
			port = server.Port
		} else {
			port = 22
		}
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return false, time.Since(start), err
	}
	_ = conn.Close()
	return true, time.Since(start), nil
}

// resolveSSHDestination uses `ssh -G <alias>` to extract HostName and Port from the user's SSH config.
// Returns host, port, ok where ok=false if resolution failed.
func resolveSSHDestination(alias string) (string, int, bool) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", 0, false
	}
	cmd := exec.Command("ssh", "-G", alias)
	out, err := cmd.Output()
	if err != nil {
		return "", 0, false
	}
	host := ""
	port := 0
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "hostname ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				host = parts[1]
			}
		}
		if strings.HasPrefix(line, "port ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if p, err := strconv.Atoi(parts[1]); err == nil {
					port = p
				}
			}
		}
	}
	if host == "" {
		host = alias
	}
	if port == 0 {
		port = 22
	}
	return host, port, true
}

func mergeIdentityFiles(existing []string, incoming []string) []string {
	if len(incoming) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing))
	for _, v := range existing {
		seen[v] = struct{}{}
	}
	for _, v := range incoming {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		existing = append(existing, v)
		seen[v] = struct{}{}
	}
	return existing
}

func matchSessionForServer(server domain.Server, session activeSSHSession) bool {
	if session.alias != "" {
		if strings.EqualFold(session.alias, server.Alias) {
			return true
		}
		for _, alias := range server.Aliases {
			if strings.EqualFold(session.alias, alias) {
				return true
			}
		}
	}
	if session.host != "" && server.Host != "" {
		return strings.EqualFold(session.host, server.Host)
	}
	return false
}

func killPID(pid int) error {
	proc, findErr := os.FindProcess(pid)
	if findErr != nil {
		return findErr
	}
	return proc.Signal(syscall.SIGTERM)
}

func mergeForwardSpecs(existing []string, incoming []string) []string {
	if len(incoming) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing))
	for _, v := range existing {
		seen[v] = struct{}{}
	}
	for _, v := range incoming {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		existing = append(existing, v)
		seen[v] = struct{}{}
	}
	return existing
}

func matchesServerQuery(server domain.Server, query string) bool {
	fields := []string{
		strings.ToLower(server.Host),
		strings.ToLower(server.User),
		strings.ToLower(server.Alias),
	}
	for _, tag := range server.Tags {
		fields = append(fields, strings.ToLower(tag))
	}
	if len(server.Aliases) > 0 {
		for _, alias := range server.Aliases {
			fields = append(fields, strings.ToLower(alias))
		}
	}

	for _, field := range fields {
		if strings.Contains(field, query) {
			return true
		}
	}
	return false
}

func (s *serverService) listActiveSSHSessions() ([]activeSSHSession, error) {
	cmd := exec.Command("ps", "-ax", "-o", "pid=", "-o", "comm=", "-o", "args=")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	sessions := make([]activeSSHSession, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		pid, comm, args := splitPSLine(line)
		if comm != "ssh" {
			continue
		}
		parts := strings.Fields(args)
		if len(parts) == 0 {
			continue
		}
		session := parseSSHArgs(parts)
		if session.alias == "" {
			continue
		}
		session.pid = pid
		sessions = append(sessions, session)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

func splitPSLine(line string) (pid int, comm string, args string) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, "", ""
	}
	if n, err := strconv.Atoi(fields[0]); err == nil {
		pid = n
	}
	comm = fields[1]
	start := strings.Index(line, comm)
	if start == -1 {
		if len(fields) > 2 {
			args = strings.Join(fields[2:], " ")
		}
		return pid, comm, strings.TrimSpace(args)
	}
	args = strings.TrimSpace(line[start+len(comm):])
	return pid, comm, args
}

func parseSSHArgs(args []string) activeSSHSession {
	if len(args) == 0 {
		return activeSSHSession{}
	}
	user := ""
	port := 0
	dest := ""
	identityFiles := make([]string, 0)
	localForward := make([]string, 0)
	remoteForward := make([]string, 0)
	dynamicForward := make([]string, 0)

	start := 1
	if args[0] != "ssh" {
		start = 0
	}
	for i := start; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if i+1 < len(args) {
				dest = args[i+1]
			}
			break
		}
		if strings.HasPrefix(arg, "-") {
			if sshOptionConsumesValue(arg) {
				val := ""
				if len(arg) > 2 {
					val = arg[2:]
				} else if i+1 < len(args) {
					val = args[i+1]
					i++
				}
				switch {
				case strings.HasPrefix(arg, "-p"):
					if n, err := strconv.Atoi(val); err == nil {
						port = n
					}
				case strings.HasPrefix(arg, "-l"):
					if val != "" {
						user = val
					}
				case strings.HasPrefix(arg, "-i"):
					if val != "" {
						identityFiles = append(identityFiles, val)
					}
				case strings.HasPrefix(arg, "-L"):
					if val != "" {
						localForward = append(localForward, val)
					}
				case strings.HasPrefix(arg, "-R"):
					if val != "" {
						remoteForward = append(remoteForward, val)
					}
				case strings.HasPrefix(arg, "-D"):
					if val != "" {
						dynamicForward = append(dynamicForward, val)
					}
				case strings.HasPrefix(arg, "-o"):
					lowerVal := strings.ToLower(val)
					if strings.HasPrefix(lowerVal, "user=") {
						user = val[len("user="):]
					}
					if strings.HasPrefix(lowerVal, "port=") {
						if n, err := strconv.Atoi(val[len("port="):]); err == nil {
							port = n
						}
					}
					if strings.HasPrefix(lowerVal, "identityfile=") {
						identity := val[len("identityfile="):]
						if identity != "" {
							identityFiles = append(identityFiles, identity)
						}
					}
					if strings.HasPrefix(lowerVal, "localforward=") {
						spec := val[len("localforward="):]
						if spec != "" {
							localForward = append(localForward, spec)
						}
					}
					if strings.HasPrefix(lowerVal, "remoteforward=") {
						spec := val[len("remoteforward="):]
						if spec != "" {
							remoteForward = append(remoteForward, spec)
						}
					}
					if strings.HasPrefix(lowerVal, "dynamicforward=") {
						spec := val[len("dynamicforward="):]
						if spec != "" {
							dynamicForward = append(dynamicForward, spec)
						}
					}
				}
			}
			continue
		}
		dest = arg
		break
	}

	if dest == "" {
		return activeSSHSession{}
	}

	host := dest
	if at := strings.LastIndex(dest, "@"); at > -1 {
		if user == "" {
			user = dest[:at]
		}
		host = dest[at+1:]
	}
	if host == "" {
		host = "unknown"
	}
	if port == 0 {
		port = 22
	}

	return activeSSHSession{
		alias:          dest,
		host:           host,
		user:           user,
		port:           port,
		identityFiles:  identityFiles,
		localForward:   localForward,
		remoteForward:  remoteForward,
		dynamicForward: dynamicForward,
	}
}

func sshOptionConsumesValue(opt string) bool {
	base := opt
	if len(opt) > 2 && strings.HasPrefix(opt, "-") && !strings.HasPrefix(opt, "--") {
		base = opt[:2]
	}
	switch base {
	case "-p", "-l", "-i", "-o", "-F", "-b", "-c", "-D", "-E", "-e", "-I", "-J", "-L", "-m", "-O", "-Q", "-R", "-S", "-W", "-w":
		return true
	default:
		return false
	}
}
