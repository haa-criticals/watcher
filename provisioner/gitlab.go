package provisioner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

type Config struct {
	BaseUrl   string
	Token     string
	ProjectId int64
	Ref       string
	Variables map[string]string
}

type Gitlab struct {
	config *Config
}

func (g *Gitlab) Create(ctx context.Context) error {
	values := g.buildValues()
	values.Add("ACTION", "create")
	return g.doRequest(ctx, values)
}

func (g *Gitlab) buildValues() url.Values {
	values := url.Values{
		"ref":   {g.config.Ref},
		"token": {g.config.Token},
	}

	for k, v := range g.config.Variables {
		values.Add(fmt.Sprintf("Variables[%s]", k), v)
	}
	return values
}

func (g *Gitlab) Destroy(ctx context.Context) error {
	values := g.buildValues()
	values.Add("ACTION", "destroy")
	return g.doRequest(ctx, values)
}

func (g *Gitlab) doRequest(ctx context.Context, values url.Values) error {
	endPoint := fmt.Sprintf("%s/api/v4/projects/%d/trigger/pipeline", g.config.BaseUrl, g.config.ProjectId)
	req, err := http.NewRequestWithContext(ctx, "POST", endPoint, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("could not close gitlab response's body: %v", err)
		}
	}(res.Body)

	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("could not trigger pipeline status code: %d, reason: %s", res.StatusCode, string(b))
	}
	return nil
}
