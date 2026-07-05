package sshnet

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/motoryang/velo-ssh/internal/transfer"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type testSecretStore map[string]string

func (s testSecretStore) Get(ref string) (string, error) {
	value, ok := s[ref]
	if !ok {
		return "", fmt.Errorf("missing secret %s", ref)
	}
	return value, nil
}

func (s testSecretStore) Set(ref, value string) error {
	s[ref] = value
	return nil
}

func (s testSecretStore) Delete(ref string) error {
	delete(s, ref)
	return nil
}

type localSSHServer struct {
	t             *testing.T
	addr          string
	root          string
	user          string
	password      string
	hostSigner    ssh.Signer
	authorizedKey ssh.PublicKey
	listener      net.Listener
	done          chan struct{}
	windowChanges chan ptyWindow
}

type ptyWindow struct {
	rows int
	cols int
}

func startLocalSSHServer(t *testing.T, opts func(*localSSHServer)) *localSSHServer {
	t.Helper()
	hostSigner := testSigner(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen local ssh: %v", err)
	}
	srv := &localSSHServer{
		t:             t,
		addr:          ln.Addr().String(),
		root:          t.TempDir(),
		user:          "testuser",
		password:      "testpass",
		hostSigner:    hostSigner,
		listener:      ln,
		done:          make(chan struct{}),
		windowChanges: make(chan ptyWindow, 8),
	}
	if opts != nil {
		opts(srv)
	}
	go srv.serve()
	t.Cleanup(srv.Close)
	return srv
}

func (s *localSSHServer) Close() {
	_ = s.listener.Close()
	select {
	case <-s.done:
	case <-time.After(2 * time.Second):
		s.t.Fatalf("local ssh server did not stop")
	}
}

func (s *localSSHServer) serve() {
	defer close(s.done)
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(meta ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if meta.User() == s.user && string(password) == s.password {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected")
		},
		PublicKeyCallback: func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if meta.User() == s.user && s.authorizedKey != nil && bytes.Equal(key.Marshal(), s.authorizedKey.Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("public key rejected")
		},
	}
	cfg.AddHostKey(s.hostSigner)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn, cfg)
	}
}

func (s *localSSHServer) handleConn(conn net.Conn, cfg *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		_ = conn.Close()
		return
	}
	go s.handleGlobalRequests(reqs)
	go func() {
		defer sshConn.Close()
		for newChannel := range chans {
			if newChannel.ChannelType() != "session" {
				_ = newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
				continue
			}
			channel, requests, err := newChannel.Accept()
			if err != nil {
				continue
			}
			go s.handleSession(channel, requests)
		}
	}()
}

func (s *localSSHServer) handleGlobalRequests(reqs <-chan *ssh.Request) {
	for req := range reqs {
		if req.WantReply {
			_ = req.Reply(req.Type == "keepalive@vssh", nil)
		}
	}
}

func (s *localSSHServer) handleSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	for req := range requests {
		switch req.Type {
		case "subsystem":
			var payload struct{ Name string }
			ssh.Unmarshal(req.Payload, &payload)
			if payload.Name != "sftp" {
				_ = req.Reply(false, nil)
				continue
			}
			_ = req.Reply(true, nil)
			server, err := sftp.NewServer(channel, sftp.WithServerWorkingDirectory(s.root))
			if err == nil {
				_ = server.Serve()
				_ = server.Close()
			}
			_ = channel.Close()
			return
		case "pty-req":
			_ = req.Reply(true, nil)
		case "window-change":
			win := parseWindowChange(req.Payload)
			select {
			case s.windowChanges <- win:
			default:
			}
		case "shell":
			_ = req.Reply(true, nil)
			go func() {
				_, _ = io.Copy(io.Discard, channel)
				_ = channel.Close()
			}()
		default:
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
		}
	}
}

func parseWindowChange(payload []byte) ptyWindow {
	if len(payload) < 8 {
		return ptyWindow{}
	}
	return ptyWindow{
		cols: int(binary.BigEndian.Uint32(payload[0:4])),
		rows: int(binary.BigEndian.Uint32(payload[4:8])),
	}
}

func testSigner(t *testing.T) ssh.Signer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}
	return signer
}

func writePrivateKey(t *testing.T, path string, key *rsa.PrivateKey) {
	t.Helper()
	data := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
}

func testRSAKeyPair(t *testing.T) (*rsa.PrivateKey, ssh.Signer) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}
	return key, signer
}

func (s *localSSHServer) configServer(authType string) config.Server {
	host, portString, err := net.SplitHostPort(s.addr)
	if err != nil {
		s.t.Fatalf("split address: %v", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		s.t.Fatalf("parse port: %v", err)
	}
	return config.Server{
		ID:          "local",
		Name:        "Local Integration",
		Env:         "test",
		Host:        host,
		Port:        port,
		User:        s.user,
		AuthType:    authType,
		PasswordRef: "password",
	}
}

func integrationSettings() config.Settings {
	settings := config.DefaultSettings()
	settings.KnownHostsPolicy = config.HostKeyInsecure
	settings.KeepAliveSeconds = 60
	return settings
}

func connectPasswordClient(t *testing.T, srv *localSSHServer) (*Client, *sftp.Client) {
	t.Helper()
	client := NewClient(integrationSettings(), testSecretStore{"password": srv.password})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Connect(ctx, srv.configServer(config.AuthPassword)); err != nil {
		t.Fatalf("connect password client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	sftpClient, err := client.OpenSFTP(ctx)
	if err != nil {
		t.Fatalf("open sftp: %v", err)
	}
	return client, sftpClient
}

func TestIntegrationPasswordAuthAndSFTP(t *testing.T) {
	server := startLocalSSHServer(t, nil)
	_, sftpClient := connectPasswordClient(t, server)
	if err := writeSFTPFile(sftpClient, "password.txt", []byte("password ok")); err != nil {
		t.Fatalf("write sftp file: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(server.root, "password.txt"))
	if err != nil {
		t.Fatalf("read server file: %v", err)
	}
	if string(got) != "password ok" {
		t.Fatalf("server file content = %q", got)
	}
}

func TestIntegrationKeyAuthAndSFTP(t *testing.T) {
	privateKey, signer := testRSAKeyPair(t)
	server := startLocalSSHServer(t, func(s *localSSHServer) {
		s.authorizedKey = signer.PublicKey()
	})
	keyPath := filepath.Join(t.TempDir(), "id_rsa")
	writePrivateKey(t, keyPath, privateKey)
	client := NewClient(integrationSettings(), testSecretStore{})
	srvConfig := server.configServer(config.AuthKey)
	srvConfig.PasswordRef = ""
	srvConfig.KeyPath = keyPath
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Connect(ctx, srvConfig); err != nil {
		t.Fatalf("connect key client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	sftpClient, err := client.OpenSFTP(ctx)
	if err != nil {
		t.Fatalf("open sftp: %v", err)
	}
	if err := writeSFTPFile(sftpClient, "key.txt", []byte("key ok")); err != nil {
		t.Fatalf("write sftp file: %v", err)
	}
}

func TestIntegrationKnownHostsMismatchRejection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	server := startLocalSSHServer(t, nil)
	wrongSigner := testSigner(t)
	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")
	if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0o700); err != nil {
		t.Fatalf("create known_hosts dir: %v", err)
	}
	if err := os.WriteFile(knownHostsPath, []byte(knownhosts.Line([]string{server.addr}, wrongSigner.PublicKey())+"\n"), 0o600); err != nil {
		t.Fatalf("write known_hosts: %v", err)
	}
	settings := config.DefaultSettings()
	settings.KnownHostsPolicy = config.HostKeyAsk
	client := NewClient(settings, testSecretStore{"password": server.password})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := client.Connect(ctx, server.configServer(config.AuthPassword))
	if err == nil {
		_ = client.Close()
		t.Fatal("expected changed host key error")
	}
	var changed *ChangedHostKeyError
	if !errors.As(err, &changed) {
		t.Fatalf("expected ChangedHostKeyError, got %T: %v", err, err)
	}
}

func TestIntegrationAtomicUploadFinalizesAfterSuccess(t *testing.T) {
	server := startLocalSSHServer(t, nil)
	_, sftpClient := connectPasswordClient(t, server)
	localPath := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(localPath, []byte("final content"), 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
	}
	var tmpPath string
	err := transfer.AtomicUpload(sftpClient, localPath, "upload.txt", "success", nil, func(path string) {
		tmpPath = path
	}, nil, nil)
	if err != nil {
		t.Fatalf("atomic upload: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(server.root, "upload.txt"))
	if err != nil {
		t.Fatalf("read uploaded file: %v", err)
	}
	if string(got) != "final content" {
		t.Fatalf("uploaded content = %q", got)
	}
	if _, err := os.Stat(filepath.Join(server.root, tmpPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected temp upload path removed, stat err=%v", err)
	}
}

func TestIntegrationFailedUploadDoesNotTruncateExistingRemoteTarget(t *testing.T) {
	server := startLocalSSHServer(t, nil)
	_, sftpClient := connectPasswordClient(t, server)
	targetPath := filepath.Join(server.root, "existing.txt")
	if err := os.WriteFile(targetPath, []byte("original"), 0o644); err != nil {
		t.Fatalf("write existing remote target: %v", err)
	}
	localPath := filepath.Join(t.TempDir(), "replacement.txt")
	if err := os.WriteFile(localPath, []byte("replacement"), 0o644); err != nil {
		t.Fatalf("write local replacement: %v", err)
	}
	canceled := make(chan struct{})
	close(canceled)
	err := transfer.AtomicUpload(sftpClient, localPath, "existing.txt", "canceled", nil, nil, canceled, nil)
	if err == nil {
		t.Fatal("expected canceled upload error")
	}
	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read existing remote target: %v", err)
	}
	if string(got) != "original" {
		t.Fatalf("existing target content changed to %q", got)
	}
	if _, err := os.Stat(filepath.Join(server.root, transfer.TempRemotePath("existing.txt", "canceled"))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected canceled upload temp path removed, stat err=%v", err)
	}
}

func TestIntegrationFailedDownloadDoesNotCreateCorruptFinalLocalFile(t *testing.T) {
	server := startLocalSSHServer(t, nil)
	_, sftpClient := connectPasswordClient(t, server)
	if err := os.WriteFile(filepath.Join(server.root, "remote.txt"), []byte("remote content"), 0o644); err != nil {
		t.Fatalf("write remote file: %v", err)
	}
	localPath := filepath.Join(t.TempDir(), "download.txt")
	canceled := make(chan struct{})
	close(canceled)
	err := transfer.AtomicDownload(sftpClient, "remote.txt", localPath, "canceled", nil, nil, canceled, nil)
	if err == nil {
		t.Fatal("expected canceled download error")
	}
	if _, err := os.Stat(localPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no final local file, stat err=%v", err)
	}
	if _, err := os.Stat(transfer.TempLocalPath(localPath, "canceled")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected canceled download temp path removed, stat err=%v", err)
	}
}

func TestIntegrationKeepAliveStopsAfterClientClose(t *testing.T) {
	server := startLocalSSHServer(t, nil)
	settings := integrationSettings()
	settings.KeepAliveSeconds = 1
	client := NewClient(settings, testSecretStore{"password": server.password})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Connect(ctx, server.configServer(config.AuthPassword)); err != nil {
		t.Fatalf("connect client: %v", err)
	}
	client.mu.Lock()
	done := client.keepaliveDone
	client.mu.Unlock()
	if done == nil {
		t.Fatal("expected keepalive done channel")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("keepalive did not stop after client close")
	}
}

func TestIntegrationResizeTriggersPTYWindowChange(t *testing.T) {
	server := startLocalSSHServer(t, nil)
	client := NewClient(integrationSettings(), testSecretStore{"password": server.password})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Connect(ctx, server.configServer(config.AuthPassword)); err != nil {
		t.Fatalf("connect client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	if _, err := client.OpenShell(ctx, PtySize{Width: 80, Height: 24}); err != nil {
		t.Fatalf("open shell: %v", err)
	}
	if err := client.WindowChange(40, 120); err != nil {
		t.Fatalf("window change: %v", err)
	}
	select {
	case got := <-server.windowChanges:
		if got.rows != 40 || got.cols != 120 {
			t.Fatalf("window change = rows %d cols %d, want rows 40 cols 120", got.rows, got.cols)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not receive window-change request")
	}
}

func writeSFTPFile(client *sftp.Client, path string, data []byte) error {
	f, err := client.Create(path)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
