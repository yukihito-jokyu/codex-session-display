package main

import (
	"bufio"
	"codex-session-display/internal/utils/logger"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

const singleInstanceAddress = "127.0.0.1:58385"

type singleInstanceServer struct {
	listener net.Listener
	onOpen   func(filePath string)
}

func newSingleInstanceServer(listener net.Listener, onOpen func(filePath string)) *singleInstanceServer {
	return &singleInstanceServer{
		listener: listener,
		onOpen:   onOpen,
	}
}

func (s *singleInstanceServer) serve(ctx context.Context) {
	go func() {
		<-ctx.Done()
		_ = s.listener.Close()
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			logger.Warn("Single instance accept failed", "error", err)
			continue
		}

		go s.handleConn(conn)
	}
}

func (s *singleInstanceServer) handleConn(conn net.Conn) {
	defer conn.Close()

	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		logger.Warn("Failed to read forwarded session path", "error", err)
		return
	}

	filePath := strings.TrimSpace(line)
	if filePath == "" {
		return
	}

	s.onOpen(filePath)
}

func acquireSingleInstanceListener() (net.Listener, error) {
	return net.Listen("tcp", singleInstanceAddress)
}

func extractSessionFileArg(args []string) string {
	if len(args) < 2 {
		return ""
	}
	return args[1]
}

func forwardSessionFilePath(address, filePath string) error {
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		return fmt.Errorf("failed to dial main instance: %w", err)
	}
	defer conn.Close()

	if _, err := fmt.Fprintln(conn, filePath); err != nil {
		return fmt.Errorf("failed to forward session path: %w", err)
	}

	return nil
}
