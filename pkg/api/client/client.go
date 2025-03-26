package client

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

var DefaultTimeout = 10 * time.Second
var DefaultConcurrencyLimit = 5

// Client wraps http.Client with rate limiting and concurrency control
type Client struct {
	client     *http.Client
	semChan    chan struct{}
	maxRetries int
	retryDelay time.Duration
}

// New creates a new API client with the given concurrency limit
func New(maxConcurrent int) *Client {
	return &Client{
		client: &http.Client{
			Timeout: DefaultTimeout,
		},
		semChan:    make(chan struct{}, maxConcurrent),
		maxRetries: 3,
		retryDelay: 1 * time.Second,
	}
}

// Do executes request with rate limiting and concurrency control
func (c *Client) Do(req *http.Request) (*http.Response, []byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay * time.Duration(attempt))
			// Создаем новый request для каждой попытки
			newReq := req.Clone(req.Context())
			if req.Body != nil {
				body, _ := req.GetBody()
				newReq.Body = body
			}
			req = newReq
		}

		c.semChan <- struct{}{}
		resp, body, err := c.doRequest(req)
		<-c.semChan

		if err == nil {
			// Проверяем HTTP статус
			if resp.StatusCode < 500 {
				if resp.StatusCode == 429 { // Rate limit
					if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
						if delay, err := time.ParseDuration(retryAfter + "s"); err == nil {
							time.Sleep(delay)
							continue
						}
					}
				}
				return resp, body, nil
			}
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}
		lastErr = err
	}
	return nil, nil, fmt.Errorf("all retries failed: %v", lastErr)
}

func (c *Client) doRequest(req *http.Request) (*http.Response, []byte, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}

	return resp, body, nil
}
