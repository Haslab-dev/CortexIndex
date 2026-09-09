package feedback

import (
	"context"
	"testing"
	"time"
)

func TestAppendListIdempotent(t *testing.T) {
	root := t.TempDir()
	e := Event{ID: "e1", Provider: "taste", Kind: Accept, Target: "pref", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	if err := Append(root, e); err != nil {
		t.Fatal(err)
	}
	if err := Append(root, e); err != nil {
		t.Fatal(err)
	}
	rows, err := List(root, "", 0)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%#v err=%v", rows, err)
	}
	_ = context.Background()
}
