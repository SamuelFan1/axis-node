package axisclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type RegisterNodeRequest struct {
	UUID              string `json:"uuid"`
	Hostname          string `json:"hostname"`
	ManagementAddress string `json:"management_address"`
	Region            string `json:"region"`
	Status            string `json:"status"`
}

type RegisterNodeResponse struct {
	Message string `json:"message"`
	Node    struct {
		UUID              string `json:"uuid"`
		Hostname          string `json:"hostname"`
		ManagementAddress string `json:"management_address"`
		Region            string `json:"region"`
		Status            string `json:"status"`
	} `json:"node"`
	Error string `json:"error"`
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) RegisterNode(req RegisterNodeRequest) (*RegisterNodeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal register request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/nodes/register", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build register request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send register request: %w", err)
	}
	defer resp.Body.Close()

	var parsed RegisterNodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode register response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if parsed.Error != "" {
			return nil, fmt.Errorf(parsed.Error)
		}
		return nil, fmt.Errorf("axis server returned status %d", resp.StatusCode)
	}

	return &parsed, nil
}
