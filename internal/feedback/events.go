package feedback

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Kind string

const (
	Accept      Kind = "accept"
	Reject      Kind = "reject"
	Edit        Kind = "edit"
	Merge       Kind = "merge"
	TaskOutcome Kind = "task_outcome"
)

type Event struct {
	ID        string            `json:"id"`
	Provider  string            `json:"provider"`
	Kind      Kind              `json:"kind"`
	Target    string            `json:"target"`
	Scope     string            `json:"scope,omitempty"`
	TaskID    string            `json:"task_id,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
	Timestamp string            `json:"timestamp"`
	Source    string            `json:"source,omitempty"`
	Note      string            `json:"note,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

func (e Event) Validate() error {
	if e.ID == "" || strings.ContainsAny(e.ID, " \t\r\n") {
		return fmt.Errorf("feedback event requires a safe id")
	}
	if e.Provider == "" || e.Target == "" {
		return fmt.Errorf("feedback event requires provider and target")
	}
	switch e.Kind {
	case Accept, Reject, Edit, Merge, TaskOutcome:
	default:
		return fmt.Errorf("unsupported feedback kind %q", e.Kind)
	}
	if e.Timestamp == "" {
		return fmt.Errorf("feedback event requires timestamp")
	}
	if _, err := time.Parse(time.RFC3339, e.Timestamp); err != nil {
		return fmt.Errorf("invalid feedback timestamp")
	}
	if len(e.Note) > 4096 {
		return fmt.Errorf("feedback note exceeds 4096 bytes")
	}
	return nil
}

func Decode(data []byte) (Event, error) {
	var e Event
	if err := json.Unmarshal(data, &e); err != nil {
		return e, err
	}
	return e, e.Validate()
}
