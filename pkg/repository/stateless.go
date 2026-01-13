package repository

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/matrixhub-ai/matrixhub/internal/utils"
)

func (r *Repository) Stateless(ctx context.Context, output io.Writer, input io.Reader, service string, advertise bool) error {
	base, dir := filepath.Split(r.repoPath)

	var args []string

	if advertise {
		// Write packet-line formatted header
		if _, err := output.Write(packetLine(fmt.Sprintf("# service=%s\n", service))); err != nil {
			return err
		}
		if _, err := output.Write([]byte("0000")); err != nil {
			return err
		}

		args = []string{
			"--stateless-rpc", "--advertise-refs", dir,
		}
	} else {
		args = []string{
			"--stateless-rpc", dir,
		}
	}
	cmd := utils.Command(ctx, service, args...)
	cmd.Dir = base
	cmd.Stdin = input
	cmd.Stdout = output
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// packetLine formats a string as a git packet-line.
func packetLine(s string) []byte {
	return []byte(fmt.Sprintf("%04x%s", len(s)+4, s))
}
