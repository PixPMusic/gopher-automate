package actions

import (
	"fmt"
	"strconv"
	"time"
)

// SleepHandler handles Sleep execution logic
type SleepHandler struct{}

func (h *SleepHandler) IsSupported() bool {
	return true
}

func (h *SleepHandler) Execute(code string) (string, error) {
	seconds, err := h.parseDuration(code)
	if err != nil {
		return "", err
	}

	time.Sleep(time.Duration(seconds * float64(time.Second)))
	return fmt.Sprintf("Slept for %.2f seconds", seconds), nil
}

func (h *SleepHandler) Validate(code string) error {
	_, err := h.parseDuration(code)
	return err
}

func (h *SleepHandler) parseDuration(code string) (float64, error) {
	if code == "" {
		return 0, fmt.Errorf("empty duration")
	}
	val, err := strconv.ParseFloat(code, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", code)
	}
	if val < 0 {
		return 0, fmt.Errorf("duration cannot be negative")
	}
	return val, nil
}
