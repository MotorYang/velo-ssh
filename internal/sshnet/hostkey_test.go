package sshnet

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestHostKeyAskMissingKnownHostsReturnsUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".ssh", "known_hosts")
	cb, err := hostKeyCallbackAt("ask", path)
	if err != nil {
		t.Fatal(err)
	}
	key := testPublicKey(t)
	err = cb("example.com:22", testAddr("example.com:22"), key)
	var unknown *UnknownHostKeyError
	if !errors.As(err, &unknown) {
		t.Fatalf("error = %T %v, want UnknownHostKeyError", err, err)
	}
	if unknown.Fingerprint != ssh.FingerprintSHA256(key) {
		t.Fatalf("fingerprint = %q", unknown.Fingerprint)
	}
	if unknown.KnownHostsLine == "" {
		t.Fatal("expected known_hosts line")
	}
}

func TestHostKeyAskUnknownExistingKnownHostsReturnsUnknown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(path, []byte(knownhosts.Line([]string{"other.example.com:22"}, testPublicKey(t))+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cb, err := hostKeyCallbackAt("ask", path)
	if err != nil {
		t.Fatal(err)
	}
	err = cb("example.com:22", testAddr("example.com:22"), testPublicKey(t))
	var unknown *UnknownHostKeyError
	if !errors.As(err, &unknown) {
		t.Fatalf("error = %T %v, want UnknownHostKeyError", err, err)
	}
}

func TestHostKeyAskChangedKnownHostsReturnsChanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(path, []byte(knownhosts.Line([]string{"example.com:22"}, testPublicKey(t))+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cb, err := hostKeyCallbackAt("ask", path)
	if err != nil {
		t.Fatal(err)
	}
	err = cb("example.com:22", testAddr("example.com:22"), testPublicKey(t))
	var changed *ChangedHostKeyError
	if !errors.As(err, &changed) {
		t.Fatalf("error = %T %v, want ChangedHostKeyError", err, err)
	}
}

func TestHostKeyStrictMissingKnownHostsFails(t *testing.T) {
	_, err := hostKeyCallbackAt("strict", filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected strict missing known_hosts to fail")
	}
}

func TestHostKeyInsecureAcceptsAnyKey(t *testing.T) {
	cb, err := hostKeyCallbackAt("insecure", filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if err := cb("example.com:22", testAddr("example.com:22"), testPublicKey(t)); err != nil {
		t.Fatalf("insecure callback error = %v", err)
	}
}

func TestAcceptHostKeyCreatesKnownHosts(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".ssh", "known_hosts")
	key := testPublicKey(t)
	unknown := newUnknownHostKeyError("example.com:22", testAddr("example.com:22"), key, path)
	if err := AcceptHostKey(unknown); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("known_hosts mode = %o, want 0600", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf(".ssh mode = %o, want 0700", dirInfo.Mode().Perm())
	}
	cb, err := hostKeyCallbackAt("ask", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cb("example.com:22", testAddr("example.com:22"), key); err != nil {
		t.Fatalf("accepted key did not match: %v", err)
	}
}

type testAddr string

func (a testAddr) Network() string { return "tcp" }
func (a testAddr) String() string  { return string(a) }

func testPublicKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

var _ net.Addr = testAddr("")
