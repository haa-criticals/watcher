package monitor

type Option func(monitor *Monitor)

func WithNotifier(notifier Notifier) Option {
	return func(m *Monitor) {
		m.notifier = notifier
	}
}

func WithErrorHandler(errorHandler ErrorHandler) Option {
	return func(m *Monitor) {
		m.errorHandler = errorHandler
	}
}
