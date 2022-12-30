package provisioner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type pipeline struct {
	Id             int       `json:"id"`
	Iid            int       `json:"iid"`
	ProjectId      int       `json:"project_id"`
	Sha            string    `json:"sha"`
	Ref            string    `json:"ref"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	WebUrl         string    `json:"web_url"`
	StartedAt      time.Time `json:"started_at"`
	FinishedAt     time.Time `json:"finished_at"`
	Duration       float64   `json:"duration"`
	QueuedDuration float64   `json:"queued_duration"`
}

type Config struct {
	BaseUrl   string
	Token     string
	ProjectId int64
	Ref       string
	Variables map[string]string
	Watch     bool
}

type Gitlab struct {
	config *Config
}

func (g *Gitlab) Create(ctx context.Context) error {
	values := g.buildValues()
	values.Add("variables[ACTION]", "create")
	pipeline, err := g.doRequest(ctx, values)
	if err != nil {
		return err
	}

	if g.config.Watch {
		return g.watchPipeline(ctx, pipeline)
	}
	return nil
}

func (g *Gitlab) buildValues() url.Values {
	values := url.Values{
		"ref":   {g.config.Ref},
		"token": {g.config.Token},
	}

	for k, v := range g.config.Variables {
		values.Add(fmt.Sprintf("variables[%s]", k), v)
	}
	return values
}

func (g *Gitlab) Destroy(ctx context.Context) error {
	values := g.buildValues()
	values.Add("variables[ACTION]", "destroy")
	pipeline, err := g.doRequest(ctx, values)
	if err != nil {
		return err
	}

	if g.config.Watch {
		return g.watchPipeline(ctx, pipeline)
	}
	return nil
}

func (g *Gitlab) doRequest(ctx context.Context, values url.Values) (*pipeline, error) {
	endPoint := fmt.Sprintf("%s/api/v4/projects/%d/trigger/pipeline", strings.TrimRight(g.config.BaseUrl, "/"), g.config.ProjectId)
	req, err := http.NewRequestWithContext(ctx, "POST", endPoint, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("could not close gitlab response's body: %v", err)
		}
	}(res.Body)

	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("could not trigger pipeline status code: %d, reason: %s", res.StatusCode, string(b))
	}

	pipeline := &pipeline{}
	if err = json.NewDecoder(res.Body).Decode(&pipeline); err != nil {
		return nil, fmt.Errorf("could not decode pipeline response: %v", err)
	}
	return pipeline, nil
}

func (g *Gitlab) watchPipeline(ctx context.Context, pipeline *pipeline) error {
	log.Printf("Operation Status: %s", pipeline.Status)
	log.Printf("Details URL: %s", pipeline.WebUrl)
	log.Printf("Operation Created at %v", pipeline.CreatedAt)
	log.Printf("Start watching Operation...")

	tick := time.NewTicker(time.Second * 30)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			pipeline, err := g.pipelineInfo(ctx, pipeline.Id)
			if err != nil {
				return err
			}
			if !pipeline.FinishedAt.IsZero() {
				log.Printf("Operation finished status: %s", pipeline.Status)
				return nil
			}
			log.Printf("Operation status: %s, Duration %s", pipeline.Status, time.Now().Sub(pipeline.CreatedAt).Truncate(time.Second))
		}
	}
}

func (g *Gitlab) pipelineInfo(ctx context.Context, id int) (*pipeline, error) {
	endpoint := fmt.Sprintf("%s/api/v4/projects/%d/pipelines/%d", strings.TrimRight(g.config.BaseUrl, "/"), g.config.ProjectId, id)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("could not close gitlab response's body: %v", err)
		}
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("could not get pipeline status code: %d, reason: %s", res.StatusCode, string(b))
	}

	pipeline := &pipeline{}
	if err = json.NewDecoder(res.Body).Decode(&pipeline); err != nil {
		return nil, fmt.Errorf("could not decode pipeline response: %v", err)
	}
	return pipeline, nil
}
