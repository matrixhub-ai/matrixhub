// Copyright The MatrixHub Authors.
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

// Package clamav is a minimal clamd client (INSTREAM scanning + version
// query). It speaks the clamd socket protocol directly — no subprocesses are
// spawned and no untrusted content is ever executed.
package clamav

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Config configures the clamd connection.
type Config struct {
	// Socket is a unix socket path (preferred) — empty means use TCPServer.
	Socket string
	// TCPServer is host:port when Socket is empty.
	TCPServer string
	// MaxScanBytes caps the streamed size (clamd's StreamMaxLength should be
	// configured to at least this value).
	MaxScanBytes int64
	// ConnTimeout / IOTimeout bound the connection and the whole session.
	ConnTimeout time.Duration
	IOTimeout   time.Duration
}

// DefaultConfig returns sane defaults (local unix socket).
func DefaultConfig(socket string) Config {
	return Config{
		Socket:       socket,
		MaxScanBytes: 2 << 30, // 2 GiB
		ConnTimeout:  5 * time.Second,
		IOTimeout:    10 * time.Minute,
	}
}

// ErrUnavailable wraps every transport-level failure so callers can map it to
// "scanner unavailable" (task failed / policy-controlled admission) instead of
// a clean verdict.
var ErrUnavailable = errors.New("clamd unavailable")

// Result of one INSTREAM scan.
type Result struct {
	// Clean is true when clamd answered "OK".
	Clean bool
	// Signature is the clamav signature name when a threat was found.
	Signature string
	// Verdict source string, e.g. "clamav/1.5.4/28135".
	Version string
}

// Client is a clamd client. It is safe for sequential use; the scan executor
// serializes access per worker.
type Client struct {
	cfg Config
}

func NewClient(cfg Config) *Client { return &Client{cfg: cfg} }

func (c *Client) dial() (net.Conn, error) {
	if c.cfg.Socket != "" {
		return net.DialTimeout("unix", c.cfg.Socket, c.cfg.ConnTimeout)
	}
	if c.cfg.TCPServer == "" {
		return nil, fmt.Errorf("%w: no socket or tcp server configured", ErrUnavailable)
	}
	return net.DialTimeout("tcp", c.cfg.TCPServer, c.cfg.ConnTimeout)
}

func (c *Client) session(cmd string) (net.Conn, *bufio.Reader, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if err := conn.SetDeadline(time.Now().Add(c.cfg.IOTimeout)); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if _, err := conn.Write([]byte(cmd)); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return conn, bufio.NewReader(conn), nil
}

// Ping checks daemon availability ("zPING\0" → "PONG").
func (c *Client) Ping() error {
	conn, r, err := c.session("zPING\x00")
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	resp, err := r.ReadString('\x00')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if !strings.Contains(resp, "PONG") {
		return fmt.Errorf("%w: unexpected ping response %q", ErrUnavailable, resp)
	}
	return nil
}

// Version returns "ClamAV <engine>/<db date or signature count>..." which is
// used as the scanner rule version.
func (c *Client) Version() (string, error) {
	conn, r, err := c.session("zVERSION\x00")
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()
	resp, err := r.ReadString('\x00')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return strings.TrimSpace(strings.TrimRight(resp, "\x00\n")), nil
}

// ScanStream streams r to clamd via INSTREAM with chunked writes bounded by
// MaxScanBytes. Oversized inputs return an error (mapped by the caller to a
// per-file "skipped: exceeds scanner size limit" finding, never as clean).
func (c *Client) ScanStream(r io.Reader) (Result, error) {
	conn, br, err := c.session("zINSTREAM\x00")
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = conn.Close() }()

	var sent int64
	buf := make([]byte, 64<<10)
	chunk := make([]byte, 4+len(buf))
	for {
		n, rerr := r.Read(buf)
		if n > 0 {
			sent += int64(n)
			if sent > c.cfg.MaxScanBytes {
				// terminate the session politely with a zero chunk
				binary.BigEndian.PutUint32(chunk[:4], 0)
				if _, werr := conn.Write(chunk[:4]); werr == nil {
					_, _ = br.ReadString('\n')
				}
				return Result{}, fmt.Errorf("input exceeds clamd scan limit %d bytes", c.cfg.MaxScanBytes)
			}
			binary.BigEndian.PutUint32(chunk[:4], uint32(n))
			copy(chunk[4:], buf[:n])
			if _, werr := conn.Write(chunk[:4+n]); werr != nil {
				return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, werr)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return Result{}, fmt.Errorf("read input: %w", rerr)
		}
	}
	// zero-length chunk terminates the stream
	binary.BigEndian.PutUint32(chunk[:4], 0)
	if _, werr := conn.Write(chunk[:4]); werr != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, werr)
	}
	// z-commands answer "<result>\x00" with no trailing newline; tolerate
	// both terminators.
	var sb strings.Builder
	for {
		c, rerr := br.ReadByte()
		if rerr != nil {
			if !errors.Is(rerr, io.EOF) && sb.Len() == 0 {
				return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, rerr)
			}
			break
		}
		if c == 0x00 || c == '\n' {
			break
		}
		sb.WriteByte(c)
	}
	resp := strings.TrimSpace(sb.String())
	switch {
	case strings.HasSuffix(resp, " OK"):
		return Result{Clean: true}, nil
	case strings.HasSuffix(resp, " FOUND"):
		sig := strings.TrimSuffix(strings.TrimSuffix(resp, " FOUND"), "stream: ")
		return Result{Clean: false, Signature: sig}, nil
	case strings.Contains(resp, "INSTREAM size limit exceeded"):
		return Result{}, fmt.Errorf("clamd stream size limit exceeded")
	default:
		return Result{}, fmt.Errorf("%w: unexpected clamd response %q", ErrUnavailable, resp)
	}
}

// MaxScanBytes exposes the configured stream cap.
func (c *Client) MaxScanBytes() int64 { return c.cfg.MaxScanBytes }
