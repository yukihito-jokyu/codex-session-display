//go:build !production

package logger

import (
	"testing"
)

func TestInitLogger(t *testing.T) {
	oldHomeFn := userHomeDirFn
	defer func() { userHomeDirFn = oldHomeFn }()

	tempDir := t.TempDir()

	tests := []struct {
		name   string
		homeFn func() (string, error)
	}{
		{
			name: "success initialization in dev environment",
			homeFn: func() (string, error) {
				return tempDir, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userHomeDirFn = tt.homeFn
			cleanup := InitLogger()
			if cleanup == nil {
				t.Fatal("InitLogger returned nil cleanup")
			}
			cleanup()
		})
	}
}
