package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// fakeAPI records requests and serves canned responses keyed by "METHOD path".
type fakeAPI struct {
	t        *testing.T
	handlers map[string]func(w http.ResponseWriter, r *http.Request, body []byte)
	calls    []string
}

func newFakeAPI(t *testing.T) (*fakeAPI, *RESTClient) {
	f := &fakeAPI{t: t, handlers: map[string]func(http.ResponseWriter, *http.Request, []byte){}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		key := r.Method + " " + r.URL.Path
		f.calls = append(f.calls, key+"?"+r.URL.RawQuery)
		h, ok := f.handlers[key]
		if !ok {
			t.Errorf("unexpected request %s", key)
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
		h(w, r, body)
	}))
	t.Cleanup(srv.Close)
	return f, NewREST(srv.URL, "test-key")
}

func TestListDatasetExamples_FollowsCursorAndStripsSystemFields(t *testing.T) {
	api, c := newFakeAPI(t)
	api.handlers["GET /v2/datasets/ds1/examples"] = func(w http.ResponseWriter, r *http.Request, _ []byte) {
		if r.URL.Query().Get("cursor") == "" {
			fmt.Fprint(w, `{"examples":[{"id":"e1","created_at":"t","updated_at":"t","annotations":[],"question":"Q1","answer":"A1"}],
				"pagination":{"has_more":true,"next_cursor":"next"}}`)
			return
		}
		fmt.Fprint(w, `{"examples":[{"id":"e2","created_at":"t","updated_at":"t","question":"Q2","score":3}],
			"pagination":{"has_more":false}}`)
	}

	got, err := c.ListDatasetExamples(context.Background(), "ds1")
	if err != nil {
		t.Fatal(err)
	}

	want := []DatasetExample{
		{ID: "e1", Fields: map[string]string{"question": "Q1", "answer": "A1"}},
		{ID: "e2", Fields: map[string]string{"question": "Q2", "score": "3"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListDatasetExamples() = %+v, want %+v", got, want)
	}
}

func TestGetDataset_NotFoundReturnsSentinel(t *testing.T) {
	api, c := newFakeAPI(t)
	api.handlers["GET /v2/datasets/gone"] = func(w http.ResponseWriter, _ *http.Request, _ []byte) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"status":404,"title":"Not Found"}`)
	}

	_, err := c.GetDataset(context.Background(), "gone")

	if !errors.Is(err, ErrDatasetNotFound) {
		t.Fatalf("GetDataset() error = %v, want ErrDatasetNotFound", err)
	}
}

func TestDataset_LatestVersionIDPicksNewestCreatedAt(t *testing.T) {
	ds := Dataset{Versions: []DatasetVersion{
		{ID: "v2", CreatedAt: "2026-10-02T00:00:00Z"},
		{ID: "v3", CreatedAt: "2026-10-03T00:00:00Z"},
		{ID: "v1", CreatedAt: "2026-10-01T00:00:00Z"},
	}}

	if got := ds.LatestVersionID(); got != "v3" {
		t.Fatalf("LatestVersionID() = %q, want v3", got)
	}
}

func TestInsertDatasetExamples_SendsBatchesOf1000(t *testing.T) {
	api, c := newFakeAPI(t)
	var batchSizes []int
	api.handlers["POST /v2/datasets/ds1/examples"] = func(w http.ResponseWriter, _ *http.Request, body []byte) {
		var req struct {
			Examples []map[string]string `json:"examples"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		batchSizes = append(batchSizes, len(req.Examples))
		fmt.Fprint(w, `{}`)
	}
	rows := make([]map[string]string, 2500)
	for i := range rows {
		rows[i] = map[string]string{"question": fmt.Sprint(i)}
	}

	if err := c.InsertDatasetExamples(context.Background(), "ds1", rows); err != nil {
		t.Fatal(err)
	}

	if want := []int{1000, 1000, 500}; !reflect.DeepEqual(batchSizes, want) {
		t.Fatalf("batch sizes = %v, want %v", batchSizes, want)
	}
}

func TestUpdateDatasetExamples_SendsIDWithFieldsInPlace(t *testing.T) {
	api, c := newFakeAPI(t)
	var got map[string]interface{}
	api.handlers["PATCH /v2/datasets/ds1/examples"] = func(w http.ResponseWriter, _ *http.Request, body []byte) {
		_ = json.Unmarshal(body, &got)
		fmt.Fprint(w, `{}`)
	}

	err := c.UpdateDatasetExamples(context.Background(), "ds1", []DatasetExample{
		{ID: "e1", Fields: map[string]string{"question": "Q1", "answer": "Paris"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]interface{}{
		"examples": []interface{}{map[string]interface{}{"id": "e1", "question": "Q1", "answer": "Paris"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PATCH body = %v, want %v (no new_version, so the update is in place)", got, want)
	}
}

func TestDeleteDatasetExamples_ErrorsWhenNotCompleted(t *testing.T) {
	api, c := newFakeAPI(t)
	var got map[string]interface{}
	api.handlers["DELETE /v2/datasets/ds1/examples"] = func(w http.ResponseWriter, _ *http.Request, body []byte) {
		_ = json.Unmarshal(body, &got)
		fmt.Fprint(w, `{"completed":false,"deleted_example_ids":[],"not_deleted_example_ids":["e1"]}`)
	}

	err := c.DeleteDatasetExamples(context.Background(), "ds1", "v1", []string{"e1"})

	if err == nil {
		t.Fatal("DeleteDatasetExamples() error = nil, want error for incomplete delete")
	}
	want := map[string]interface{}{"dataset_version_id": "v1", "example_ids": []interface{}{"e1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DELETE body = %v, want %v", got, want)
	}
}

func TestCreateDataset_SendsSpaceNameAndRows(t *testing.T) {
	api, c := newFakeAPI(t)
	var got map[string]interface{}
	api.handlers["POST /v2/datasets"] = func(w http.ResponseWriter, _ *http.Request, body []byte) {
		_ = json.Unmarshal(body, &got)
		fmt.Fprint(w, `{"id":"ds1","name":"quiz","space_id":"sp1","versions":[{"id":"v1","created_at":"t"}]}`)
	}

	ds, err := c.CreateDataset(context.Background(), "sp1", "quiz", []map[string]string{{"question": "Q1"}})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]interface{}{"space_id": "sp1", "name": "quiz", "examples": []interface{}{map[string]interface{}{"question": "Q1"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("POST body = %v, want %v", got, want)
	}
	if ds.ID != "ds1" || ds.LatestVersionID() != "v1" {
		t.Fatalf("CreateDataset() = %+v, want id ds1 with version v1", ds)
	}
}

func TestRenameDataset_PatchesName(t *testing.T) {
	api, c := newFakeAPI(t)
	var got map[string]interface{}
	api.handlers["PATCH /v2/datasets/ds1"] = func(w http.ResponseWriter, _ *http.Request, body []byte) {
		_ = json.Unmarshal(body, &got)
		fmt.Fprint(w, `{}`)
	}

	if err := c.RenameDataset(context.Background(), "ds1", "new-name"); err != nil {
		t.Fatal(err)
	}

	if want := map[string]interface{}{"name": "new-name"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("PATCH body = %v, want %v", got, want)
	}
}

func TestDeleteDataset_AlreadyGoneIsNotAnError(t *testing.T) {
	api, c := newFakeAPI(t)
	api.handlers["DELETE /v2/datasets/ds1"] = func(w http.ResponseWriter, _ *http.Request, _ []byte) {
		w.WriteHeader(http.StatusNotFound)
	}

	if err := c.DeleteDataset(context.Background(), "ds1"); err != nil {
		t.Fatalf("DeleteDataset() error = %v, want nil for 404", err)
	}
}

func TestDatasetExampleUnmarshal_NullColumnIsUnset(t *testing.T) {
	// Arize fills a column missing from a row with null.
	var ex DatasetExample
	if err := json.Unmarshal([]byte(`{"id":"e1","q":"b","x":null}`), &ex); err != nil {
		t.Fatal(err)
	}
	if _, ok := ex.Fields["x"]; ok {
		t.Fatalf("Fields = %v, want null column x left out", ex.Fields)
	}
	if ex.Fields["q"] != "b" {
		t.Fatalf("Fields = %v, want q = b", ex.Fields)
	}
}

func TestDatasetExampleMarshal_ClearedColumnIsNull(t *testing.T) {
	b, err := json.Marshal(DatasetExample{ID: "e1", Fields: map[string]string{"q": "a"}, Clear: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"id":"e1","q":"a","x":null}` {
		t.Fatalf("JSON = %s, want x sent as null (null clears a column in place)", b)
	}
}
