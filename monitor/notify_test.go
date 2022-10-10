package monitor

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockEndpoint struct {
	server   *httptest.Server
	lastBeat time.Time
}

func (m *mockEndpoint) start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/monitor/beat", func(w http.ResponseWriter, r *http.Request) {
		m.lastBeat = time.Now()
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			log.Println(err)
		}
	})
	m.server = httptest.NewServer(mux)
}

func (m *mockEndpoint) stop() {
	m.server.Close()
}

func (m *mockEndpoint) baseURL() string {
	return m.server.URL
}

func TestBeat(t *testing.T) {
	t.Run("Should send heart beat to endpoint", func(t *testing.T) {
		m := &mockEndpoint{}
		m.start()
		n := &defaultNotifier{}
		start := time.Now()

		err := n.beat(fmt.Sprintf("%s/monitor/beat", m.baseURL()))
		assert.NoError(t, err)
		assert.NotNil(t, m.lastBeat)
		assert.True(t, m.lastBeat.After(start), "last beat should be after start")
		m.stop()
	})
}
