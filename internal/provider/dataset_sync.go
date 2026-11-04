package provider

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

// exampleDiff is the set of changes that makes a dataset's examples match the
// configured rows, with rows matched to examples by the key column.
type exampleDiff struct {
	Insert []map[string]string
	Update []client.DatasetExample
	Delete []string
}

// diffExamples compares the dataset's examples with the configured rows, which
// are keyed by their key column value.
func diffExamples(keyColumn string, existing []client.DatasetExample, desired map[string]map[string]string) exampleDiff {
	byKey := make(map[string]client.DatasetExample, len(existing))
	for _, ex := range existing {
		byKey[ex.Fields[keyColumn]] = ex
	}

	keys := make([]string, 0, len(desired))
	for k := range desired {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var d exampleDiff
	for _, key := range keys {
		row := desired[key]
		ex, ok := byKey[key]
		switch {
		case !ok:
			d.Insert = append(d.Insert, row)
		case !reflect.DeepEqual(ex.Fields, row):
			// The update is a patch: columns no longer in the row are cleared.
			var clear []string
			for col := range ex.Fields {
				if _, ok := row[col]; !ok {
					clear = append(clear, col)
				}
			}
			sort.Strings(clear)
			d.Update = append(d.Update, client.DatasetExample{ID: ex.ID, Fields: row, Clear: clear})
		}
	}
	for _, ex := range existing {
		if _, ok := desired[ex.Fields[keyColumn]]; !ok {
			d.Delete = append(d.Delete, ex.ID)
		}
	}
	return d
}

// reservedExampleColumns are fields the API sets on every example.
var reservedExampleColumns = []string{"id", "created_at", "updated_at", "annotations"}

// validateExampleRows checks that every row carries its map key in the key
// column and has no column that collides with a system field.
func validateExampleRows(keyColumn string, rows map[string]map[string]string) error {
	keys := make([]string, 0, len(rows))
	for k := range rows {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		row := rows[key]
		if key == "" {
			return fmt.Errorf("an example has an empty %s; every example needs one", keyColumn)
		}
		for _, col := range reservedExampleColumns {
			if _, ok := row[col]; ok {
				return fmt.Errorf("example %q: column %q is reserved by Arize", key, col)
			}
		}
		val, ok := row[keyColumn]
		if !ok {
			return fmt.Errorf("example %q has no %q column", key, keyColumn)
		}
		if val != key {
			return fmt.Errorf("example %q: %s is %q; it must equal the map key", key, keyColumn, val)
		}
	}
	return nil
}

// examplesByKey keys the dataset's examples by their key column value, the
// shape of the examples attribute.
func examplesByKey(keyColumn string, fetched []client.DatasetExample) (map[string]map[string]string, error) {
	out := make(map[string]map[string]string, len(fetched))
	count := make(map[string]int, len(fetched))
	var dups []string
	for _, ex := range fetched {
		key := ex.Fields[keyColumn]
		count[key]++
		if count[key] == 2 {
			dups = append(dups, key)
		}
		out[key] = ex.Fields
	}
	if len(dups) > 0 {
		msgs := make([]string, len(dups))
		for i, k := range dups {
			msgs[i] = fmt.Sprintf("%d examples in Arize have %s %q", count[k], keyColumn, k)
		}
		return nil, fmt.Errorf("%s; remove the duplicates in Arize so each %s is unique", strings.Join(msgs, ", "), keyColumn)
	}
	return out, nil
}

// parseDatasetImportID splits "<dataset_id>:<key_column>". Dataset IDs are
// base64, which never contains a colon.
func parseDatasetImportID(importID string) (datasetID, keyColumn string, err error) {
	datasetID, keyColumn, ok := strings.Cut(importID, ":")
	if !ok || datasetID == "" || keyColumn == "" {
		return "", "", fmt.Errorf("expected import ID \"<dataset_id>:<key_column>\", e.g. \"RGF0YXNldDox:question\"; got %q", importID)
	}
	return datasetID, keyColumn, nil
}
