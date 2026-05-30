//go:build !production

package logger

import (
	"testing"
)

func TestInitLogger(t *testing.T) {
	oldHomeFn := userHomeDirFn
	defer func() { userHomeDirFn = oldHomeFn }()

	tempDir := t.TempDir()
	userHomeDirFn = func() (string, error) {
		return tempDir, nil
	}

	cleanup := InitLogger()
	if cleanup == nil {
		t.Fatal("InitLogger returned nil cleanup")
	}
	cleanup()
}
