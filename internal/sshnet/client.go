package sshnet

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Client struct {
	settings config.Settings
	secrets  config.SecretStore
	server   config.Server

	mu              sync.Mutex
	ssh             *ssh.Client
	sftp            *sftp.Client
	shell           *Shell
	cancelKeepalive context.CancelFunc
	keepaliveDone   chan struct{}
	stale           bool
}

type PtySize struct {
	Width  int
	Height int
}

func NewClient(settings config.Settings, secrets config.SecretStore) *Client {
	return &Client{settings: settings, secrets: secrets}
}

func (c *Client) Connect(ctx context.Context, srv config.Server) error {
	if err := config.ValidateServer(srv); err != nil {
		return err
	}
	auth, err := c.authMethods(srv)
	if err != nil {
		return err
	}
	cb, err := hostKeyCallback(c.settings.KnownHostsPolicy)
	if err != nil {
		return err
	}
	sshCfg := &ssh.ClientConfig{
		User:            srv.User,
		Auth:            auth,
		HostKeyCallback: cb,
		Timeout:         15 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", srv.Host, srv.Port)
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("connect %s: %w", addr, err)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, sshCfg)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("ssh handshake %s: %w", addr, err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	c.mu.Lock()
	c.server = srv
	c.ssh = client
	c.stale = false
	c.mu.Unlock()
	c.startKeepAlive(context.Background())
	return nil
}

func (c *Client) OpenSFTP(ctx context.Context) (*sftp.Client, error) {
	c.mu.Lock()
	stale := c.stale
	if c.sftp != nil {
		defer c.mu.Unlock()
		if stale {
			return nil, fmt.Errorf("stale connection: ssh keepalive failed")
		}
		return c.sftp, nil
	}
	client := c.ssh
	c.mu.Unlock()
	if stale {
		return nil, fmt.Errorf("stale connection: ssh keepalive failed")
	}
	if client == nil {
		return nil, fmt.Errorf("open sftp: ssh client is not connected")
	}
	type result struct {
		c   *sftp.Client
		err error
	}
	ch := make(chan result, 1)
	go func() {
		sc, err := sftp.NewClient(client)
		ch <- result{c: sc, err: err}
	}()
	select {
	case <-ctx.Done():
		go func() {
			res := <-ch
			if res.c != nil {
				_ = res.c.Close()
			}
		}()
		return nil, ctx.Err()
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		c.mu.Lock()
		c.sftp = res.c
		c.mu.Unlock()
		return res.c, nil
	}
}

func (c *Client) OpenShell(ctx context.Context, size PtySize) (*Shell, error) {
	c.mu.Lock()
	client := c.ssh
	stale := c.stale
	c.mu.Unlock()
	if stale {
		return nil, fmt.Errorf("stale connection: ssh keepalive failed")
	}
	if client == nil {
		return nil, fmt.Errorf("open shell: ssh client is not connected")
	}
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	if size.Width <= 0 {
		size.Width = 80
	}
	if size.Height <= 0 {
		size.Height = 24
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", size.Height, size.Width, modes); err != nil {
		_ = session.Close()
		return nil, err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		return nil, err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		return nil, err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		_ = session.Close()
		return nil, err
	}
	if err := session.Shell(); err != nil {
		_ = session.Close()
		return nil, err
	}
	sh := &Shell{session: session, stdin: stdin, stdout: stdout, stderr: stderr, help: EscapeHelpWithLanguage(c.settings.Language)}
	c.mu.Lock()
	c.shell = sh
	c.mu.Unlock()
	return sh, nil
}

func (c *Client) Reconnect(ctx context.Context) error {
	c.mu.Lock()
	srv := c.server
	c.mu.Unlock()
	c.Close()
	return c.Connect(ctx, srv)
}

func (c *Client) Close() error {
	c.keepaliveCancel()
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	if c.shell != nil {
		err = c.shell.Close()
		c.shell = nil
	}
	if c.sftp != nil {
		if e := c.sftp.Close(); err == nil {
			err = e
		}
		c.sftp = nil
	}
	if c.ssh != nil {
		if e := c.ssh.Close(); err == nil {
			err = e
		}
		c.ssh = nil
	}
	return err
}

func (c *Client) WindowChange(height, width int) error {
	c.mu.Lock()
	sh := c.shell
	c.mu.Unlock()
	if sh == nil {
		return nil
	}
	return sh.WindowChange(height, width)
}

func (c *Client) Stale() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stale
}

func (c *Client) RunInteractiveShell(ctx context.Context, in *os.File, out, errOut *os.File, onLocal func(EscapeResult)) error {
	c.mu.Lock()
	sh := c.shell
	c.mu.Unlock()
	var err error
	if sh == nil || sh.Closed() {
		sh, err = c.OpenShell(ctx, PtySize{Width: 80, Height: 24})
		if err != nil {
			return err
		}
	}
	defer func() {
		if sh.Closed() {
			c.mu.Lock()
			if c.shell == sh {
				c.shell = nil
			}
			c.mu.Unlock()
		}
	}()
	err = sh.Run(in, out, errOut, onLocal)
	if sh.Closed() {
		c.mu.Lock()
		if c.shell == sh {
			c.shell = nil
		}
		c.mu.Unlock()
	}
	return err
}

func (c *Client) authMethods(srv config.Server) ([]ssh.AuthMethod, error) {
	switch srv.AuthType {
	case config.AuthPassword:
		if srv.PasswordRef == "" {
			return nil, fmt.Errorf("password auth requires passwordRef")
		}
		secret, err := c.secrets.Get(srv.PasswordRef)
		if err != nil {
			return nil, fmt.Errorf("read password from keyring: %w", err)
		}
		return []ssh.AuthMethod{ssh.Password(secret)}, nil
	case config.AuthKey:
		keyPath := expandHome(srv.KeyPath)
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, err
		}
		var signer ssh.Signer
		if srv.PassphraseRef != "" {
			passphrase, err := c.secrets.Get(srv.PassphraseRef)
			if err != nil {
				return nil, fmt.Errorf("read passphrase from keyring: %w", err)
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
			if err != nil {
				return nil, err
			}
		} else {
			signer, err = ssh.ParsePrivateKey(key)
			if err != nil {
				return nil, err
			}
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	case config.AuthAgent:
		sock := os.Getenv("SSH_AUTH_SOCK")
		if sock == "" {
			return nil, fmt.Errorf("SSH_AUTH_SOCK is not set")
		}
		conn, err := net.Dial("unix", sock)
		if err != nil {
			return nil, err
		}
		return []ssh.AuthMethod{ssh.PublicKeysCallback(agent.NewClient(conn).Signers)}, nil
	default:
		return nil, fmt.Errorf("unsupported auth type %q", srv.AuthType)
	}
}

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	usr, err := user.Current()
	if err != nil {
		return path
	}
	if path == "~" {
		return usr.HomeDir
	}
	return filepath.Join(usr.HomeDir, strings.TrimPrefix(path, "~/"))
}
