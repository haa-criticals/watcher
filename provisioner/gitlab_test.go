package provisioner

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type gitlabRequest struct {
	projectId string
	action    string
	variables url.Values
}

type gitlabMock struct {
	server   *http.Server
	requests []gitlabRequest
}

func (g *gitlabMock) init() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/", func(w http.ResponseWriter, r *http.Request) {
		projectId := strings.Split(r.URL.Path, "/")[4]
		g.requests = append(g.requests, gitlabRequest{
			projectId: projectId,
			action:    r.FormValue("ACTION"),
			variables: r.PostForm,
		})
		w.WriteHeader(http.StatusCreated)
	})
	g.server = &http.Server{Addr: ":3580", Handler: mux}
	return g.server.ListenAndServe()
}

func TestGitlabProvider(t *testing.T) {
	g := &gitlabMock{}
	go func() {
		t.Logf("starting server")
		if err := g.init(); err != nil {
			t.Errorf("failed to start server: %s", err)
			return
		}
	}()

	t.Run("Should call create pipeline", func(t *testing.T) {
		m := New(&Config{
			BaseUrl:   "http://localhost:3580",
			Token:     "token",
			ProjectId: 10,
			Ref:       "main",
			Variables: map[string]string{
				"requestId": "1",
			},
		})

		if err := m.Create(context.Background()); err != nil {
			t.Errorf("failed to create pipeline: %s", err)
			return
		}

		if len(g.requests) < 1 {
			t.Errorf("no requests received")
			return
		}

		requestReceived := false
		for _, request := range g.requests {
			if request.variables.Get("Variables[requestId]") != "1" {
				continue
			}

			requestReceived = true

			if request.action != "create" {
				t.Errorf("expected action to be create, got %s", request.action)
			}
			if request.projectId != "10" {
				t.Errorf("expected proejectId to be 10, got %s", request.projectId)
			}
			if request.variables.Get("ref") != "main" {
				t.Errorf("expected ref to be main, got %s", request.variables.Get("ref"))
			}

			if request.variables.Get("token") != "token" {
				t.Errorf("expected token to be token, got %s", request.variables.Get("token"))
			}
			break
		}

		if !requestReceived {
			t.Errorf("Request was not received")
		}
	})

	t.Run("Should call destroy pipeline", func(t *testing.T) {
		m := New(&Config{
			BaseUrl:   "http://localhost:3580",
			Token:     "token",
			ProjectId: 10,
			Ref:       "main",
			Variables: map[string]string{
				"requestId": "2",
			},
		})

		if err := m.Destroy(context.Background()); err != nil {
			t.Errorf("failed to destroy pipeline: %s", err)
			return
		}

		if len(g.requests) < 1 {
			t.Errorf("no requests received")
			return
		}

		requestReceived := false
		for _, request := range g.requests {
			if request.variables.Get("Variables[requestId]") != "2" {
				continue
			}

			requestReceived = true

			if request.action != "destroy" {
				t.Errorf("expected action to be destroy, got %s", request.action)
			}
			if request.projectId != "10" {
				t.Errorf("expected proejectId to be 10, got %s", request.projectId)
			}
			if request.variables.Get("ref") != "main" {
				t.Errorf("expected ref to be main, got %s", request.variables.Get("ref"))
			}

			if request.variables.Get("token") != "token" {
				t.Errorf("expected token to be token, got %s", request.variables.Get("token"))
			}
			break
		}

		if !requestReceived {
			t.Errorf("Request was not received")
		}
	})

}
