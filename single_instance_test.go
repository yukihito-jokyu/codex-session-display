package main

import (
	"bufio"
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractSessionFileArg(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "session.jsonl")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "returns first extra argument",
			args: []string{"app", validPath},
			want: validPath,
		},
		{
			name: "returns empty when no extra argument",
			args: []string{"app"},
			want: "",
		},
		{
			name: "ignores later arguments",
			args: []string{"app", validPath, "ignored"},
			want: validPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSessionFileArg(tt.args)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestForwardSessionFilePath(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	received := make(chan string, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()

		line, readErr := bufio.NewReader(conn).ReadString('\n')
		if readErr != nil {
			return
		}
		received <- line[:len(line)-1]
	}()

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "sends file path",
			filePath: "/tmp/example-session.jsonl",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := forwardSessionFilePath(listener.Addr().String(), tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}

			select {
			case got := <-received:
				if got != tt.filePath {
					t.Fatalf("expected %q, got %q", tt.filePath, got)
				}
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for forwarded file path")
			}
		})
	}
}

func TestSingleInstanceServer_ServesIncomingPaths(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	received := make(chan string, 1)
	server := newSingleInstanceServer(listener, func(filePath string) {
		received <- filePath
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go server.serve(ctx)

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "receives forwarded path",
			filePath: "/tmp/from-secondary.jsonl",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := forwardSessionFilePath(listener.Addr().String(), tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}

			select {
			case got := <-received:
				if got != tt.filePath {
					t.Fatalf("expected %q, got %q", tt.filePath, got)
				}
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for server callback")
			}
		})
	}
}
