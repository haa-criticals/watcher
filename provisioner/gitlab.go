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

type Gitlab struct {
	url       string
	projectId int64
	variables map[string]string
}

func (g *Gitlab) Create(ctx context.Context) error {
	values := g.buildValues()
	values.Add("ACTION", "create")
	return g.doRequest(ctx, values)
}

func (g *Gitlab) buildValues() url.Values {
	values := url.Values{
		"ref":   {"master"},
		"token": {"token"},
	}

	for k, v := range g.variables {
		values.Add(fmt.Sprintf("variables[%s]", k), v)
	}
	return values
}

func (g *Gitlab) Destroy(ctx context.Context) error {
	values := g.buildValues()
	values.Add("ACTION", "destroy")
	return g.doRequest(ctx, values)
}

func (g *Gitlab) doRequest(ctx context.Context, values url.Values) error {
	endPoint := fmt.Sprintf("%s/api/v4/projects/%d/trigger/pipeline", g.url, g.projectId)
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
