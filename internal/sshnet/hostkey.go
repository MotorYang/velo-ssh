package sshnet

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type UnknownHostKeyError struct {
	Host           string
	Remote         string
	Fingerprint    string
	KnownHostsLine string
	KnownHostsPath string
	Key            ssh.PublicKey
}

func (e *UnknownHostKeyError) Error() string {
	return fmt.Sprintf("ssh host key verification failed for %s: host is not trusted. Fingerprint %s. Accept it only if you trust this server, then retry.", e.Host, e.Fingerprint)
}

type ChangedHostKeyError struct {
	Host           string
	Remote         string
	Fingerprint    string
	KnownHostsPath string
	Err            error
}

func (e *ChangedHostKeyError) Error() string {
	return fmt.Sprintf("ssh host key verification failed for %s: host key changed. Fingerprint %s. Check %s and verify the server before reconnecting: %v", e.Host, e.Fingerprint, e.KnownHostsPath, e.Err)
}

func (e *ChangedHostKeyError) Unwrap() error {
	return e.Err
}

func hostKeyCallback(policy string) (ssh.HostKeyCallback, error) {
	return hostKeyCallbackAt(policy, defaultKnownHostsPath())
}

func hostKeyCallbackAt(policy, knownHostsPath string) (ssh.HostKeyCallback, error) {
	if policy == "insecure" {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	cb, err := knownhosts.New(knownHostsPath)
	if err == nil {
		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			if err := cb(hostname, remote, key); err != nil {
				return classifyHostKeyError(err, hostname, remote, key, knownHostsPath)
			}
			return nil
		}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	switch policy {
	case "ask":
		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return newUnknownHostKeyError(hostname, remote, key, knownHostsPath)
		}, nil
	default:
		return nil, fmt.Errorf("known_hosts not found and host key policy is %q", policy)
	}
}

func classifyHostKeyError(err error, hostname string, remote net.Addr, key ssh.PublicKey, knownHostsPath string) error {
	var keyErr *knownhosts.KeyError
	if errors.As(err, &keyErr) {
		if len(keyErr.Want) > 0 {
			return &ChangedHostKeyError{
				Host:           hostname,
				Remote:         remoteString(remote),
				Fingerprint:    ssh.FingerprintSHA256(key),
				KnownHostsPath: knownHostsPath,
				Err:            err,
			}
		}
		return newUnknownHostKeyError(hostname, remote, key, knownHostsPath)
	}
	return fmt.Errorf("ssh host key verification failed for %s using %s: %w", hostname, knownHostsPath, err)
}

func newUnknownHostKeyError(hostname string, remote net.Addr, key ssh.PublicKey, knownHostsPath string) *UnknownHostKeyError {
	return &UnknownHostKeyError{
		Host:           hostname,
		Remote:         remoteString(remote),
		Fingerprint:    ssh.FingerprintSHA256(key),
		KnownHostsLine: knownhosts.Line([]string{hostname}, key),
		KnownHostsPath: knownHostsPath,
		Key:            key,
	}
}

func AcceptHostKey(err *UnknownHostKeyError) error {
	if err == nil {
		return fmt.Errorf("accept host key: missing host key error")
	}
	if err.KnownHostsLine == "" {
		return fmt.Errorf("accept host key for %s: missing known_hosts line", err.Host)
	}
	if err.KnownHostsPath == "" {
		err.KnownHostsPath = defaultKnownHostsPath()
	}
	if err.Key != nil {
		if cb, cbErr := knownhosts.New(err.KnownHostsPath); cbErr == nil {
			if cbErr := cb(err.Host, stringAddr(err.Remote), err.Key); cbErr == nil {
				return nil
			}
		}
	}
	if data, readErr := os.ReadFile(err.KnownHostsPath); readErr == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == strings.TrimSpace(err.KnownHostsLine) {
				return nil
			}
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	if err := os.MkdirAll(filepath.Dir(err.KnownHostsPath), 0o700); err != nil {
		return err
	}
	f, openErr := os.OpenFile(err.KnownHostsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if openErr != nil {
		return openErr
	}
	defer f.Close()
	if _, statErr := f.Stat(); statErr == nil {
		_ = os.Chmod(err.KnownHostsPath, 0o600)
	}
	if _, writeErr := fmt.Fprintln(f, err.KnownHostsLine); writeErr != nil {
		return writeErr
	}
	return nil
}

func defaultKnownHostsPath() string {
	return filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
}

func remoteString(remote net.Addr) string {
	if remote == nil {
		return ""
	}
	return remote.String()
}

type stringAddr string

func (a stringAddr) Network() string { return "tcp" }
func (a stringAddr) String() string  { return string(a) }
