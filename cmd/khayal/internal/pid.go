package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

const pidFileName = "khayal.pid"

func getPidFilePath() (string, error) {
	absPath, err := filepath.Abs(ConfigPath())
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(absPath)
	return filepath.Join(dir, pidFileName), nil
}

func PidFilePath() string {
	p, _ := getPidFilePath()
	return p
}

func WritePID() (int, error) {
	pid := os.Getpid()

	pidFile, err := getPidFilePath()
	if err != nil {
		return 0, fmt.Errorf("failed to get PID file path: %w", err)
	}

	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0600); err != nil {
		return 0, fmt.Errorf("failed to write PID file: %w", err)
	}

	return pid, nil
}

func RemovePID() error {
	pidFile, err := getPidFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	return nil
}

func GetPID() (int, error) {
	pidFile, err := getPidFilePath()
	if err != nil {
		return 0, err
	}

	data, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("khayal is not running (no PID file)")
		}
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

func IsRunning() bool {
	pid, err := GetPID()
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}
