package monitor

import (
	"fmt"
	"net/http"
)

type Notifier interface {
	Beat(watcherAddress string) error
}

type defaultNotifier struct {
}

func (n *defaultNotifier) Beat(watcherAddress string) error {
	r, err := http.NewRequest("POST", fmt.Sprintf("%s/monitor/beat", watcherAddress), nil)
	if err != nil {
		return err
	}
	_, err = http.DefaultClient.Do(r)
	return err
}
