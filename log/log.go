package log

//go:generate go run github.com/vektra/mockery/v2@latest --name=Logger --filename=logger.go --output=../mocks
type Logger interface {
	Error(err error, message string, fields map[string]any)
	Warn(message string, fields map[string]any)
	Info(message string, fields map[string]any)
	Debug(message string, fields map[string]any)
}

type EmptyLogger struct{}

func (e EmptyLogger) Error(_ error, _ string, _ map[string]any) {}

func (e EmptyLogger) Warn(_ string, _ map[string]any) {}

func (e EmptyLogger) Info(_ string, _ map[string]any) {}

func (e EmptyLogger) Debug(_ string, _ map[string]any) {}
