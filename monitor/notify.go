package monitor

import (
	"fmt"
	"net/http"
)

type Notifier interface {
	Beat(peerAddress string, address string, term int64) error
}

type defaultNotifier struct {
}

func (n *defaultNotifier) Beat(peerAddress string, _ string, _ int64) error {
	r, err := http.NewRequest("POST", fmt.Sprintf("%s/monitor/beat", peerAddress), nil)
	if err != nil {
		return err
	}
	_, err = http.DefaultClient.Do(r)
	return err
}
