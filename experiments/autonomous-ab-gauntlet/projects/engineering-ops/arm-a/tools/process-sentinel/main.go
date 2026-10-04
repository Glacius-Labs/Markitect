package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const defaultTimeout = 5 * time.Second

type observation struct {
	ExitCode int
	Elapsed time.Duration
}

func observe(ctx context.Context, executable string, args []string) (observation, error) {
	started := time.Now()
	cmd := exec.CommandContext(ctx, executable, args...)
	err := cmd.Run()
	result := observation{Elapsed: time.Since(started)}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.ExitCode = 124
		return result, nil
	}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.ExitCode = 124
		return result, nil
	}
	return result, err
}

func main() {
	timeout := flag.Duration("timeout", defaultTimeout, "maximum child-process duration")
	requestID := flag.String("request-id", "", "request identifier for diagnostics")
	flag.Parse()
	if *timeout <= 0 || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: process-sentinel [--timeout DURATION] [--request-id ID] COMMAND [ARG...]")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	result, err := observe(ctx, flag.Arg(0), flag.Args()[1:])
	if err != nil {
		if *requestID != "" {
			fmt.Fprintf(os.Stderr, "request_id=%s process_error=%v\n", *requestID, err)
		} else {
			fmt.Fprintf(os.Stderr, "process_error=%v\n", err)
		}
		os.Exit(2)
	}
	fmt.Printf("exit=%d elapsed=%s\n", result.ExitCode, result.Elapsed.Round(time.Millisecond))
	os.Exit(result.ExitCode)
}
