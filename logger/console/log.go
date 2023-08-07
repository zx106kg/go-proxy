package console

import (
	"fmt"
)

type Logger struct{}

func (l *Logger) Info(message string) {
	fmt.Printf("[INFO] %s\n", message)
}

func (l *Logger) Warn(message string) {
	fmt.Printf("[WARN] %s\n", message)
}

func NewLogger() *Logger {
	return &Logger{}
}
