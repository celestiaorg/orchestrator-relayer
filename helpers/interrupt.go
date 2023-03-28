package helpers

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	tmlog "github.com/tendermint/tendermint/libs/log"
)

// TrapSignal will listen for any OS signal and cancel the context to gracefully exit.
func TrapSignal(logger tmlog.Logger, cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)

	signal.Notify(sigCh, syscall.SIGTERM)
	signal.Notify(sigCh, syscall.SIGINT)

	sig := <-sigCh
	logger.Info("caught signal; shutting down...", "signal", sig.String())
	cancel()
}
