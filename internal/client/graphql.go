package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
}

func New(endpoint, apiKey string) *Client {
	httpClient := &http.Client{
		Transport: &authTransport{
			apiKey:    apiKey,
			roundTrip: http.DefaultTransport,
		},
	}

	return &Client{
		httpClient: httpClient,
		endpoint:   endpoint,
		apiKey:     apiKey,
	}
}

type authTransport struct {
	apiKey    string
	roundTrip http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("x-api-key", t.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return t.roundTrip.RoundTrip(req)
}

type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   interface{}       `json:"data"`
	Errors []GraphQLError    `json:"errors"`
}

type GraphQLError struct {
	Message string                 `json:"message"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *GraphQLError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s (%s)", e.Message, e.Code)
	}
	return e.Message
}

func (c *Client) execute(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	req := &GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute GraphQL request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var gqlResp GraphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %v", gqlResp.Errors[0].Error())
	}

	if err := json.Unmarshal(respBody, &gqlResp); err == nil {
		resultBytes, _ := json.Marshal(gqlResp.Data)
		if err := json.Unmarshal(resultBytes, result); err != nil {
			return fmt.Errorf("failed to parse GraphQL data: %w", err)
		}
	}

	return nil
}
