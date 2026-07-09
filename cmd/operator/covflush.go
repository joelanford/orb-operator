//go:build cover

package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/coverage"
	"syscall"
)

func init() {
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGUSR1)
		for range ch {
			dir := os.Getenv("GOCOVERDIR")
			if dir == "" {
				fmt.Fprintln(os.Stderr, "covflush: GOCOVERDIR not set, skipping")
				continue
			}
			if err := coverage.WriteCountersDir(dir); err != nil {
				fmt.Fprintf(os.Stderr, "covflush: WriteCountersDir: %v\n", err)
			}
			if err := coverage.WriteMetaDir(dir); err != nil {
				fmt.Fprintf(os.Stderr, "covflush: WriteMetaDir: %v\n", err)
			}
			fmt.Fprintln(os.Stderr, "covflush: coverage data written")
		}
	}()
}
