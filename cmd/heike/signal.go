package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type SignalHandler struct {
	ctx     context.Context
	cancel  context.CancelFunc
	sigChan chan os.Signal
	wg      sync.WaitGroup
}

func NewSignalHandler(ctx context.Context) *SignalHandler {
	ctx, cancel := context.WithCancel(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	return &SignalHandler{
		ctx:     ctx,
		cancel:  cancel,
		sigChan: sigChan,
		wg:      sync.WaitGroup{},
	}
}

func (s *SignalHandler) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-s.sigChan
		fmt.Println("\nReceived shutdown signal...")
		s.cancel()
	}()
}

func (s *SignalHandler) Wait() {
	s.wg.Wait()
}

func (s *SignalHandler) Stop() {
	signal.Stop(s.sigChan)
}
