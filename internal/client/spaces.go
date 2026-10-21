package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// DeleteSpace uses the REST API: the GraphQL deleteSpace mutation fails with
// an internal server error.
func (c *RESTClient) DeleteSpace(ctx context.Context, spaceID string) error {
	if err := c.do(ctx, http.MethodDelete, "/v2/spaces/"+url.PathEscape(spaceID), nil, nil); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete space: %w", err)
	}
	return nil
}
