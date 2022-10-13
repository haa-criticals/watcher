package monitor

import "time"

type Option func(monitor *Monitor)

func WithHeartBeat(notifier Notifier, interval time.Duration) Option {
	return func(m *Monitor) {
		m.notifier = notifier
		m.heartBeatInterval = interval
	}
}

func WithHealthCheck(endpoint string, interval time.Duration, maxFail int) Option {
	return func(m *Monitor) {
		if m.healthChecker == nil {
			m.healthChecker = &healthChecker{}
		}
		m.healthChecker.endpoint = endpoint
		m.healthChecker.interval = interval
		m.healthChecker.maxFails = maxFail
	}
}

func WithErrorHandler(errorHandler ErrorHandler) Option {
	return func(m *Monitor) {
		m.errorHandler = errorHandler
	}
}
