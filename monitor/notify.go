package monitor

import "net/http"

type Notifier interface {
	beat(endpoint string) error
}

type defaultNotifier struct {
}

func (n *defaultNotifier) beat(endpoint string) error {
	r, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return err
	}
	_, err = http.DefaultClient.Do(r)
	return err
}
