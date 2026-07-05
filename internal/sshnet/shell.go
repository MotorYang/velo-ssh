package sshnet

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	xterm "golang.org/x/term"
)

type Shell struct {
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
	help    string
	once    sync.Once
	wg      sync.WaitGroup
	mu      sync.Mutex
	out     io.Writer
	errOut  io.Writer
	outBuf  bytes.Buffer
	errBuf  bytes.Buffer
	closed  bool
}

func (s *Shell) Run(in *os.File, out, errOut *os.File, onLocal func(EscapeResult)) error {
	if xterm.IsTerminal(int(in.Fd())) {
		oldState, err := xterm.MakeRaw(int(in.Fd()))
		if err != nil {
			return fmt.Errorf("set local terminal raw mode: %w", err)
		}
		defer func() {
			_ = xterm.Restore(int(in.Fd()), oldState)
		}()
	}
	s.startPumps()
	s.attach(out, errOut)
	defer s.detach()
	help := s.help
	if help == "" {
		help = EscapeHelp()
	}
	localExit, runErr := runShellInputWithHelp(in, s.stdin, errOut, onLocal, help)
	if localExit {
		return runErr
	}
	_ = s.stdin.Close()
	_ = s.session.Wait()
	s.markClosed()
	waitGroupWithTimeout(&s.wg, 2*time.Second)
	return runErr
}

func (s *Shell) startPumps() {
	s.once.Do(func() {
		s.wg.Add(2)
		go func() {
			defer s.wg.Done()
			s.copyOutput(s.stdout, false)
		}()
		go func() {
			defer s.wg.Done()
			s.copyOutput(s.stderr, true)
		}()
	})
}

func (s *Shell) copyOutput(r io.Reader, stderr bool) {
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			s.writeOutput(buf[:n], stderr)
		}
		if err != nil {
			return
		}
	}
}

func (s *Shell) writeOutput(data []byte, stderr bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	target := s.out
	buffer := &s.outBuf
	if stderr {
		target = s.errOut
		buffer = &s.errBuf
	}
	if target != nil {
		_, _ = target.Write(data)
		return
	}
	appendLimited(buffer, data)
}

func appendLimited(buffer *bytes.Buffer, data []byte) {
	const maxBufferedShellOutput = 1 << 20
	if len(data) >= maxBufferedShellOutput {
		buffer.Reset()
		_, _ = buffer.Write(data[len(data)-maxBufferedShellOutput:])
		return
	}
	if buffer.Len()+len(data) > maxBufferedShellOutput {
		current := buffer.Bytes()
		keep := maxBufferedShellOutput - len(data)
		if keep < 0 {
			keep = 0
		}
		if keep > len(current) {
			keep = len(current)
		}
		kept := append([]byte(nil), current[len(current)-keep:]...)
		buffer.Reset()
		_, _ = buffer.Write(kept)
	}
	_, _ = buffer.Write(data)
}

func (s *Shell) attach(out, errOut io.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.out = out
	s.errOut = errOut
	if s.outBuf.Len() > 0 && out != nil {
		_, _ = out.Write(s.outBuf.Bytes())
		s.outBuf.Reset()
	}
	if s.errBuf.Len() > 0 && errOut != nil {
		_, _ = errOut.Write(s.errBuf.Bytes())
		s.errBuf.Reset()
	}
}

func (s *Shell) detach() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.out = nil
	s.errOut = nil
}

func (s *Shell) markClosed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

func (s *Shell) Closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func runShellInput(in io.Reader, remote io.Writer, errOut io.Writer, onLocal func(EscapeResult)) (bool, error) {
	return runShellInputWithHelp(in, remote, errOut, onLocal, EscapeHelp())
}

func runShellInputWithHelp(in io.Reader, remote io.Writer, errOut io.Writer, onLocal func(EscapeResult), helpText string) (bool, error) {
	buf := make([]byte, 1)
	capturing := false
	candidate := ""
	remoteLine := ""
	skipLF := false
	escapeSeq := false
	escapeSeqIntroducer := false
	escapeSeqBuffer := ""
	for {
		n, err := in.Read(buf)
		if n > 0 {
			b := buf[0]
			if skipLF && b == '\n' {
				skipLF = false
				continue
			}
			skipLF = false
			if escapeSeq {
				escapeSeqBuffer += string(b)
				if !escapeSeqIntroducer && (b == '[' || b == 'O') {
					escapeSeqIntroducer = true
					continue
				}
				if !escapeSeqIntroducer || isEscapeSequenceFinal(b) {
					if !isBracketedPasteMarker(escapeSeqBuffer) {
						_, _ = remote.Write([]byte(escapeSeqBuffer))
					}
					escapeSeq = false
					escapeSeqIntroducer = false
					escapeSeqBuffer = ""
				}
				continue
			}
			if capturing {
				if isBackspace(b) {
					if len(candidate) > 0 {
						candidate = candidate[:len(candidate)-1]
						_, _ = errOut.Write([]byte("\b \b"))
					}
					if candidate == "" {
						capturing = false
					}
					continue
				}
				candidate += string(b)
				echoLocalCapture(errOut, b)
				if isLineBreak(b) {
					skipLF = b == '\r'
					line := strings.TrimRight(candidate, "\r\n")
					res := ParseEscapeLine(normalizeLocalCandidate(line))
					if res.Local {
						if res.Command == "send" {
							_, _ = fmt.Fprintln(remote, res.Arg)
						} else {
							if res.Help || res.Unknown {
								writeLocalMessage(errOut, helpText)
							}
							if onLocal != nil {
								onLocal(res)
							}
							if shouldExitShell(res) {
								return true, nil
							}
						}
					} else {
						_, _ = remote.Write([]byte(candidate))
					}
					candidate = ""
					capturing = false
					continue
				}
				continue
			}
			if b == ':' {
				capturing = true
				candidate = ":"
				echoLocalCapture(errOut, b)
				continue
			}
			if b == 0x1b {
				escapeSeq = true
				escapeSeqIntroducer = false
				escapeSeqBuffer = string([]byte{b})
				continue
			}
			_, _ = remote.Write([]byte{b})
			if isBackspace(b) {
				if len(remoteLine) > 0 {
					remoteLine = remoteLine[:len(remoteLine)-1]
				}
				continue
			}
			remoteLine += string(b)
			if isLineBreak(b) {
				line := normalizeLocalCandidate(strings.TrimRight(remoteLine, "\r\n"))
				remoteLine = ""
				if isRemoteExitLine(line) {
					res := EscapeResult{Local: true, Command: "quit"}
					if onLocal != nil {
						onLocal(res)
					}
					return true, nil
				}
			}
			skipLF = b == '\r'
		}
		if err == io.EOF {
			if candidate != "" {
				res := ParseEscapeLine(normalizeLocalCandidate(candidate))
				if res.Local {
					if res.Command == "send" {
						_, _ = fmt.Fprintln(remote, res.Arg)
					} else if onLocal != nil {
						onLocal(res)
						if shouldExitShell(res) {
							return true, nil
						}
					}
				} else {
					_, _ = remote.Write([]byte(candidate))
				}
			}
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}
}

func isLineBreak(b byte) bool {
	return b == '\n' || b == '\r'
}

func isBackspace(b byte) bool {
	return b == 0x7f || b == 0x08
}

func echoLocalCapture(w io.Writer, b byte) {
	if isLineBreak(b) {
		_, _ = w.Write([]byte("\r\n"))
		return
	}
	if b >= 0x20 && b != 0x7f {
		_, _ = w.Write([]byte{b})
	}
}

func writeLocalMessage(w io.Writer, message string) {
	message = strings.ReplaceAll(message, "\r\n", "\n")
	message = strings.ReplaceAll(message, "\n", "\r\n")
	_, _ = fmt.Fprint(w, "\r\n"+message+"\r\n")
}

func isEscapeSequenceFinal(b byte) bool {
	return b >= 0x40 && b <= 0x7e
}

func isBracketedPasteMarker(seq string) bool {
	return seq == "\x1b[200~" || seq == "\x1b[201~"
}

func normalizeLocalCandidate(s string) string {
	s = strings.ReplaceAll(s, "\x1b[200~", "")
	s = strings.ReplaceAll(s, "\x1b[201~", "")
	return s
}

func isRemoteExitLine(line string) bool {
	fields := strings.Fields(line)
	return len(fields) > 0 && fields[0] == "exit"
}

func waitGroupWithTimeout(wg *sync.WaitGroup, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

func shouldExitShell(res EscapeResult) bool {
	switch res.Command {
	case "files", "tasks", "settings", "back", "reconnect", "quit":
		return true
	default:
		return false
	}
}

func (s *Shell) WindowChange(height, width int) error {
	return s.session.WindowChange(height, width)
}

func (s *Shell) Close() error {
	s.markClosed()
	s.detach()
	defer waitGroupWithTimeout(&s.wg, 2*time.Second)
	return s.session.Close()
}
