package logger

type Logger interface {
	Info(message string)
	Warn(message string)
}
