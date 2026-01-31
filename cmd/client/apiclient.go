package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type apiClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func newAPIClient() (*apiClient, error) {
	tok, addr, ok := checkAuth()
	if !ok {
		return nil, fmt.Errorf("not authenticated — run 'fxtunnel login' first")
	}

	webURL := DefaultServerURL
	if addr != "" {
		host := addr
		if idx := strings.Index(addr, ":"); idx != -1 {
			host = addr[:idx]
		}
		webURL = "https://" + host
	}

	return &apiClient{
		baseURL: webURL + "/api",
		token:   tok,
		http:    &http.Client{},
	}, nil
}

func (c *apiClient) do(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	return c.http.Do(req)
}

func (c *apiClient) get(path string) (*http.Response, error) {
	return c.do("GET", path, nil)
}

func (c *apiClient) post(path string, body interface{}) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.do("POST", path, strings.NewReader(string(data)))
}

func (c *apiClient) delete(path string) (*http.Response, error) {
	return c.do("DELETE", path, nil)
}

func decodeJSON[T any](resp *http.Response) (T, error) {
	var result T
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, err
	}
	return result, nil
}

func apiError(resp *http.Response) error {
	defer resp.Body.Close()
	var errResp struct {
		Error string `json:"error"`
		Code  string `json:"code,omitempty"`
	}
	json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error != "" {
		return fmt.Errorf("%s", errResp.Error)
	}
	return fmt.Errorf("server returned status %d", resp.StatusCode)
}
