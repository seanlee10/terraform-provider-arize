package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

var ErrDatasetNotFound = errors.New("dataset not found")

// maxExamplesPerRequest is the API's limit on examples in one create, insert,
// update, or delete request.
const maxExamplesPerRequest = 1000

type Dataset struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	SpaceID   string           `json:"space_id"`
	CreatedAt string           `json:"created_at"`
	UpdatedAt string           `json:"updated_at"`
	Versions  []DatasetVersion `json:"versions"`
}

type DatasetVersion struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// LatestVersionID returns the newest version, which is the one the examples
// endpoints read and write when no version is given.
func (d *Dataset) LatestVersionID() string {
	var latest DatasetVersion
	for _, v := range d.Versions {
		if v.CreatedAt >= latest.CreatedAt {
			latest = v
		}
	}
	return latest.ID
}

// DatasetExample is one row of a dataset: the system-assigned ID plus the
// user-defined columns.
type DatasetExample struct {
	ID     string
	Fields map[string]string
}

// systemExampleFields are set by the API on every example and are not columns.
var systemExampleFields = map[string]bool{"id": true, "created_at": true, "updated_at": true, "annotations": true}

func (e *DatasetExample) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	e.Fields = make(map[string]string, len(raw))
	for k, v := range raw {
		if k == "id" {
			if err := json.Unmarshal(v, &e.ID); err != nil {
				return fmt.Errorf("example id: %w", err)
			}
			continue
		}
		if systemExampleFields[k] {
			continue
		}
		// Columns are strings; anything else is kept as its JSON text.
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			s = string(v)
		}
		e.Fields[k] = s
	}
	return nil
}

func (e DatasetExample) MarshalJSON() ([]byte, error) {
	m := make(map[string]string, len(e.Fields)+1)
	for k, v := range e.Fields {
		m[k] = v
	}
	m["id"] = e.ID
	return json.Marshal(m)
}

func datasetPath(datasetID string) string {
	return "/v2/datasets/" + url.PathEscape(datasetID)
}

// CreateDataset creates a dataset with its first examples (at least one, at
// most maxExamplesPerRequest; insert the rest afterwards).
func (c *RESTClient) CreateDataset(ctx context.Context, spaceID, name string, rows []map[string]string) (*Dataset, error) {
	body := map[string]interface{}{"space_id": spaceID, "name": name, "examples": rows}
	var ds Dataset
	if err := c.do(ctx, http.MethodPost, "/v2/datasets", body, &ds); err != nil {
		return nil, fmt.Errorf("failed to create dataset: %w", err)
	}
	return &ds, nil
}

func (c *RESTClient) GetDataset(ctx context.Context, datasetID string) (*Dataset, error) {
	var ds Dataset
	if err := c.do(ctx, http.MethodGet, datasetPath(datasetID), nil, &ds); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return nil, ErrDatasetNotFound
		}
		return nil, fmt.Errorf("failed to get dataset: %w", err)
	}
	return &ds, nil
}

func (c *RESTClient) RenameDataset(ctx context.Context, datasetID, name string) error {
	if err := c.do(ctx, http.MethodPatch, datasetPath(datasetID), map[string]string{"name": name}, nil); err != nil {
		return fmt.Errorf("failed to rename dataset: %w", err)
	}
	return nil
}

func (c *RESTClient) DeleteDataset(ctx context.Context, datasetID string) error {
	if err := c.do(ctx, http.MethodDelete, datasetPath(datasetID), nil, nil); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete dataset: %w", err)
	}
	return nil
}

// ListDatasetExamples returns every example in the dataset's latest version.
func (c *RESTClient) ListDatasetExamples(ctx context.Context, datasetID string) ([]DatasetExample, error) {
	var examples []DatasetExample
	cursor := ""
	for {
		q := url.Values{"limit": {"500"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var page struct {
			Examples   []DatasetExample `json:"examples"`
			Pagination struct {
				HasMore    bool   `json:"has_more"`
				NextCursor string `json:"next_cursor"`
			} `json:"pagination"`
		}
		if err := c.do(ctx, http.MethodGet, datasetPath(datasetID)+"/examples?"+q.Encode(), nil, &page); err != nil {
			return nil, fmt.Errorf("failed to list dataset examples: %w", err)
		}
		examples = append(examples, page.Examples...)
		if !page.Pagination.HasMore || page.Pagination.NextCursor == "" {
			return examples, nil
		}
		cursor = page.Pagination.NextCursor
	}
}

// InsertDatasetExamples appends rows to the dataset's latest version.
func (c *RESTClient) InsertDatasetExamples(ctx context.Context, datasetID string, rows []map[string]string) error {
	for _, batch := range batches(rows) {
		body := map[string]interface{}{"examples": batch}
		if err := c.do(ctx, http.MethodPost, datasetPath(datasetID)+"/examples", body, nil); err != nil {
			return fmt.Errorf("failed to insert dataset examples: %w", err)
		}
	}
	return nil
}

// UpdateDatasetExamples patches examples by ID in the latest version, in place
// (no new version). Columns not sent are left unchanged.
func (c *RESTClient) UpdateDatasetExamples(ctx context.Context, datasetID string, examples []DatasetExample) error {
	for _, batch := range batches(examples) {
		body := map[string]interface{}{"examples": batch}
		if err := c.do(ctx, http.MethodPatch, datasetPath(datasetID)+"/examples", body, nil); err != nil {
			return fmt.Errorf("failed to update dataset examples: %w", err)
		}
	}
	return nil
}

// DeleteDatasetExamples removes examples from a version in place.
func (c *RESTClient) DeleteDatasetExamples(ctx context.Context, datasetID, versionID string, exampleIDs []string) error {
	for _, batch := range batches(exampleIDs) {
		body := map[string]interface{}{"dataset_version_id": versionID, "example_ids": batch}
		var resp struct {
			Completed     bool     `json:"completed"`
			NotDeletedIDs []string `json:"not_deleted_example_ids"`
		}
		if err := c.do(ctx, http.MethodDelete, datasetPath(datasetID)+"/examples", body, &resp); err != nil {
			return fmt.Errorf("failed to delete dataset examples: %w", err)
		}
		if !resp.Completed {
			return fmt.Errorf("failed to delete dataset examples: delete did not complete for %d example(s); retry the apply", len(resp.NotDeletedIDs))
		}
	}
	return nil
}

func batches[T any](items []T) [][]T {
	var out [][]T
	for len(items) > maxExamplesPerRequest {
		out = append(out, items[:maxExamplesPerRequest])
		items = items[maxExamplesPerRequest:]
	}
	if len(items) > 0 {
		out = append(out, items)
	}
	return out
}
