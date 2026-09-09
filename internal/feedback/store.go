package feedback

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Path(root string) string { return filepath.Join(root, ".cortex", "feedback", "events.jsonl") }

func Append(root string, event Event) error {
	if err := event.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o755); err != nil {
		return err
	}
	existing, err := List(root, "", 0)
	if err != nil {
		return err
	}
	for _, prior := range existing {
		if prior.ID == event.ID {
			return nil
		}
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(Path(root), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}

func List(root, kind string, limit int) ([]Event, error) {
	f, err := os.Open(Path(root))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Event
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		e, err := Decode(scanner.Bytes())
		if err != nil {
			return nil, fmt.Errorf("invalid feedback event: %w", err)
		}
		if kind == "" || string(e.Kind) == kind {
			out = append(out, e)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp < out[j].Timestamp })
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func Format(events []Event) string {
	var b strings.Builder
	b.WriteString("# Feedback Events\n\n")
	if len(events) == 0 {
		b.WriteString("_No feedback events._\n")
		return b.String()
	}
	for _, e := range events {
		b.WriteString(fmt.Sprintf("- `%s` `%s` target `%s` provider `%s` at %s", e.ID, e.Kind, e.Target, e.Provider, e.Timestamp))
		if e.Note != "" {
			b.WriteString(": " + e.Note)
		}
		b.WriteString("\n")
	}
	return b.String()
}
