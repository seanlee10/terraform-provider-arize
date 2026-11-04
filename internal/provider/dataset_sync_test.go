package provider

import (
	"reflect"
	"strings"
	"testing"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

func TestDiffExamples_ChangedAnswerIsUpdatedByExampleID(t *testing.T) {
	existing := []client.DatasetExample{
		{ID: "ex-1", Fields: map[string]string{"question": "Capital of France?", "answer": "Lyon"}},
	}
	desired := map[string]map[string]string{
		"Capital of France?": {"question": "Capital of France?", "answer": "Paris"},
	}

	got := diffExamples("question", existing, desired)

	want := exampleDiff{
		Update: []client.DatasetExample{
			{ID: "ex-1", Fields: map[string]string{"question": "Capital of France?", "answer": "Paris"}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diffExamples() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestDiffExamples_NewQuestionIsInserted(t *testing.T) {
	existing := []client.DatasetExample{
		{ID: "ex-1", Fields: map[string]string{"question": "Q1", "answer": "A1"}},
	}
	desired := map[string]map[string]string{
		"Q1": {"question": "Q1", "answer": "A1"},
		"Q2": {"question": "Q2", "answer": "A2"},
	}

	got := diffExamples("question", existing, desired)

	want := exampleDiff{Insert: []map[string]string{{"question": "Q2", "answer": "A2"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diffExamples() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestDiffExamples_MissingQuestionIsDeleted(t *testing.T) {
	existing := []client.DatasetExample{
		{ID: "ex-1", Fields: map[string]string{"question": "Q1", "answer": "A1"}},
		{ID: "ex-2", Fields: map[string]string{"question": "Q2", "answer": "A2"}},
	}
	desired := map[string]map[string]string{
		"Q1": {"question": "Q1", "answer": "A1"},
	}

	got := diffExamples("question", existing, desired)

	want := exampleDiff{Delete: []string{"ex-2"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diffExamples() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestDiffExamples_InsertsInKeyOrder(t *testing.T) {
	desired := map[string]map[string]string{
		"b": {"question": "b"}, "c": {"question": "c"}, "a": {"question": "a"},
	}

	got := diffExamples("question", nil, desired)

	want := []map[string]string{{"question": "a"}, {"question": "b"}, {"question": "c"}}
	if !reflect.DeepEqual(got.Insert, want) {
		t.Fatalf("Insert = %v, want %v (deterministic order)", got.Insert, want)
	}
}

func TestValidateExampleRows_RejectsKeyColumnNotMatchingMapKey(t *testing.T) {
	rows := map[string]map[string]string{"Q1": {"question": "Q2", "answer": "A1"}}

	err := validateExampleRows("question", rows)

	if err == nil || !strings.Contains(err.Error(), `example "Q1": question is "Q2"; it must equal the map key`) {
		t.Fatalf("validateExampleRows() error = %v, want key mismatch error", err)
	}
}

func TestValidateExampleRows_RejectsMissingKeyColumn(t *testing.T) {
	rows := map[string]map[string]string{"Q1": {"answer": "A1"}}

	err := validateExampleRows("question", rows)

	if err == nil || !strings.Contains(err.Error(), `example "Q1" has no "question" column`) {
		t.Fatalf("validateExampleRows() error = %v, want missing-column error", err)
	}
}

func TestValidateExampleRows_RejectsEmptyKey(t *testing.T) {
	rows := map[string]map[string]string{"": {"question": "", "answer": "A1"}}

	err := validateExampleRows("question", rows)

	if err == nil || !strings.Contains(err.Error(), "empty question") {
		t.Fatalf("validateExampleRows() error = %v, want empty-key error", err)
	}
}

func TestValidateExampleRows_RejectsReservedColumn(t *testing.T) {
	rows := map[string]map[string]string{"Q1": {"question": "Q1", "id": "x"}}

	err := validateExampleRows("question", rows)

	if err == nil || !strings.Contains(err.Error(), `column "id" is reserved`) {
		t.Fatalf("validateExampleRows() error = %v, want reserved-column error", err)
	}
}

func TestExamplesByKey_KeysFetchedExamplesByKeyColumn(t *testing.T) {
	fetched := []client.DatasetExample{
		{ID: "a", Fields: map[string]string{"question": "Q1", "answer": "A1"}},
		{ID: "b", Fields: map[string]string{"question": "Q2", "answer": "A2"}},
	}

	got, err := examplesByKey("question", fetched)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]map[string]string{
		"Q1": {"question": "Q1", "answer": "A1"},
		"Q2": {"question": "Q2", "answer": "A2"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("examplesByKey() = %v, want %v", got, want)
	}
}

func TestExamplesByKey_ErrorsOnDuplicateKeyInArize(t *testing.T) {
	fetched := []client.DatasetExample{
		{ID: "a", Fields: map[string]string{"question": "Q1"}},
		{ID: "b", Fields: map[string]string{"question": "Q1"}},
	}

	_, err := examplesByKey("question", fetched)

	if err == nil || !strings.Contains(err.Error(), `2 examples in Arize have question "Q1"`) {
		t.Fatalf("examplesByKey() error = %v, want duplicate error", err)
	}
}

func TestParseDatasetImportID(t *testing.T) {
	id, key, err := parseDatasetImportID("RGF0YXNldDox:question")
	if err != nil || id != "RGF0YXNldDox" || key != "question" {
		t.Fatalf("parseDatasetImportID() = %q, %q, %v; want RGF0YXNldDox, question, nil", id, key, err)
	}
	for _, bad := range []string{"RGF0YXNldDox", ":question", "RGF0YXNldDox:"} {
		if _, _, err := parseDatasetImportID(bad); err == nil {
			t.Errorf("parseDatasetImportID(%q) error = nil, want error", bad)
		}
	}
}

func TestDiffExamples_RemovedColumnIsCleared(t *testing.T) {
	existing := []client.DatasetExample{{ID: "e1", Fields: map[string]string{"q": "a", "x": "1"}}}

	d := diffExamples("q", existing, map[string]map[string]string{"a": {"q": "a"}})

	if len(d.Update) != 1 || len(d.Update[0].Clear) != 1 || d.Update[0].Clear[0] != "x" {
		t.Fatalf("Update = %+v, want e1 updated with x cleared", d.Update)
	}
}
