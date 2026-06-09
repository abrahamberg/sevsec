package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/abrahamberg/devsec/internal/contract"
)

func (c *Client) GetRuntimeEnv(ctx context.Context, project string) (map[string]string, error) {
	body := contract.RuntimeEnvRequest{
		Project:     project,
		Environment: "local",
		Reason:      "run-command",
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/runtime-env",
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("runtime env request failed: %s", resp.Status)
	}

	var result contract.RuntimeEnvResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Env, nil
}
