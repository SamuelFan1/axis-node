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
	baseURL     string
	sharedToken string
	httpClient  *http.Client
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

type ReportNodeRequest struct {
	UUID               string  `json:"uuid"`
	Hostname           string  `json:"hostname"`
	ManagementAddress  string  `json:"management_address"`
	Region             string  `json:"region"`
	Status             string  `json:"status"`
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	DiskUsagePercent   float64 `json:"disk_usage_percent"`
}

type ReportNodeResponse struct {
	Message string `json:"message"`
	Node    struct {
		UUID string `json:"UUID"`
	} `json:"node"`
	Error string `json:"error"`
}

func New(baseURL, sharedToken string) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		sharedToken: strings.TrimSpace(sharedToken),
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
	httpReq.Header.Set("X-Axis-Node-Token", c.sharedToken)

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

func (c *Client) ReportNode(req ReportNodeRequest) (*ReportNodeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal report request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/nodes/report", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build report request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Axis-Node-Token", c.sharedToken)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send report request: %w", err)
	}
	defer resp.Body.Close()

	var parsed ReportNodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode report response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if parsed.Error != "" {
			return nil, fmt.Errorf(parsed.Error)
		}
		return nil, fmt.Errorf("axis server returned status %d", resp.StatusCode)
	}

	return &parsed, nil
}
