package utils

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	log.Printf("Executing command: %s %s", name, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stderr = os.Stderr
	return cmd
}
