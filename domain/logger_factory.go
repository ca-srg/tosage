package domain

type LoggerFactory interface {
	CreateLogger(component string) Logger
}
