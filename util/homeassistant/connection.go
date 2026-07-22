package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const (
	defaultTimeout = 10 * time.Second
)

type Connection struct {
	client *http.Client
	url    string
}

type StateResponse struct {
	State      string                 `json:"state"`
	Attributes map[string]interface{} `json:"attributes"`
}

func NewConnection(url string) *Connection {
	return &Connection{
		client: request.NewClient(),
		url:    url,
	}
}

func (c *Connection) request(ctx context.Context, path string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+path, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("HA_TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("homeassistant request failed: %d", res.StatusCode)
	}

	return io.ReadAll(res.Body)
}

// CallService calls a Home Assistant service
func (c *Connection) CallService(ctx context.Context, domain, service string, target map[string]string, data map[string]interface{}) error {
	payload := map[string]interface{}{
		"target": target,
	}

	for k, v := range data {
		payload[k] = v
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/api/services/"+domain+"/"+service, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("HA_TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("homeassistant service call failed: %d", res.StatusCode)
	}

	return nil
}

// GetState retrieves the state of an entity
func (c *Connection) GetState(entity string) (StateResponse, error) {
	var res StateResponse

	data, err := c.request(context.Background(), "/api/states/"+url.QueryEscape(entity))
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(data, &res)
	return res, nil
}

// GetIntState retrieves the state of an entity as int64.
// States are parsed as float and truncated to also accept
// fractional states like "55.0" (consistent with plugin/getter.go).
func (c *Connection) GetIntState(entity string) (int64, error) {
	value, err := c.GetFloatState(entity)
	return int64(value), err
}

// GetFloatState retrieves the state of an entity as float64
func (c *Connection) GetFloatState(entity string) (float64, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseFloat(state.State, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric state '%s' for entity %s: %w", state.State, entity, err)
	}

	return value, nil
}
