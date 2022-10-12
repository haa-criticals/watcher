package monitor

import "time"

type Option func(monitor *Monitor)

func WithHeartBeat(notifier Notifier, interval time.Duration) Option {
	return func(m *Monitor) {
		m.notifier = notifier
		m.heartBeatInterval = interval
	}
}

func WithErrorHandler(errorHandler ErrorHandler) Option {
	return func(m *Monitor) {
		m.errorHandler = errorHandler
	}
}
