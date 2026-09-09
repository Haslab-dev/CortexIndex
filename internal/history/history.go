package history

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultTimeout = 5 * time.Second
	DefaultLimit   = 10
	MaxLimit       = 50
	DefaultDiffCap = 64 * 1024
)

var commitRef = regexp.MustCompile(`^[0-9A-Fa-f][0-9A-Fa-f._/-]{3,127}$`)

type Client struct {
	Root    string
	Timeout time.Duration
}

type Commit struct {
	Hash      string
	ShortHash string
	Author    string
	Date      time.Time
	Subject   string
}

type ChangedFile struct {
	Status string
	Path   string
}

type CommitDetail struct {
	Commit
	Body          string
	Files         []ChangedFile
	Diff          string
	DiffTruncated bool
}

type LogOptions struct {
	Query string
	File  string
	Since string
	Limit int
}

type ShowOptions struct {
	File     string
	WithDiff bool
	MaxBytes int
}

func New(root string) *Client {
	return &Client{Root: root, Timeout: DefaultTimeout}
}

func (c *Client) Log(ctx context.Context, o LogOptions) ([]Commit, error) {
	if err := validateLimit(o.Limit); err != nil {
		return nil, err
	}
	if err := c.validateRepo(ctx); err != nil {
		return nil, err
	}
	args := []string{"-C", c.Root, "--no-pager", "log", "--no-color", "--date=iso-strict", "--format=%H%x1f%h%x1f%an%x1f%aI%x1f%s%x1e", "--max-count=" + strconv.Itoa(normalizeLimit(o.Limit))}
	if o.Query != "" {
		args = append(args, "--grep="+o.Query, "--fixed-strings", "--regexp-ignore-case")
	}
	if o.Since != "" {
		args = append(args, "--since="+o.Since)
	}
	if o.File != "" {
		args = append(args, "--", o.File)
	}
	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseLog(out)
}

func (c *Client) Show(ctx context.Context, ref string, o ShowOptions) (CommitDetail, error) {
	if !commitRef.MatchString(ref) || strings.HasPrefix(ref, "-") {
		return CommitDetail{}, fmt.Errorf("invalid commit reference %q", ref)
	}
	if o.MaxBytes <= 0 {
		o.MaxBytes = DefaultDiffCap
	}
	if err := c.validateRepo(ctx); err != nil {
		return CommitDetail{}, err
	}
	args := []string{"-C", c.Root, "--no-pager", "show", "--no-color", "--no-ext-diff", "--format=%H%x1f%h%x1f%an%x1f%aI%x1f%s%x1f%b%x1e", "--name-status", "--no-renames"}
	if o.WithDiff {
		args = append(args, "--patch")
	}
	args = append(args, ref)
	if o.File != "" {
		args = append(args, "--", o.File)
	}
	out, err := c.run(ctx, args...)
	if err != nil {
		return CommitDetail{}, err
	}
	return parseShow(out, o.WithDiff, o.MaxBytes)
}

func (c *Client) validateRepo(ctx context.Context) error {
	out, err := c.run(ctx, "-C", c.Root, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("not a Git repository: %w", err)
	}
	if strings.TrimSpace(out) == "" {
		return errors.New("not a Git repository: empty root")
	}
	return nil
}

func (c *Client) run(parent context.Context, args ...string) (string, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if ctx.Err() != nil {
		return "", fmt.Errorf("git command timed out: %w", ctx.Err())
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git command failed: %s", msg)
	}
	return string(data), nil
}

func validateLimit(n int) error {
	if n < 0 || n > MaxLimit {
		return fmt.Errorf("limit must be between 1 and %d", MaxLimit)
	}
	return nil
}

func normalizeLimit(n int) int {
	if n == 0 {
		return DefaultLimit
	}
	return n
}

func parseLog(raw string) ([]Commit, error) {
	var out []Commit
	for _, record := range strings.Split(raw, string(rune(0x1e))) {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		fields := strings.Split(record, string(rune(0x1f)))
		if len(fields) != 5 {
			return nil, fmt.Errorf("malformed Git log record")
		}
		date, err := time.Parse(time.RFC3339, strings.TrimSpace(fields[3]))
		if err != nil {
			return nil, fmt.Errorf("malformed Git commit date: %w", err)
		}
		out = append(out, Commit{Hash: fields[0], ShortHash: fields[1], Author: fields[2], Date: date, Subject: fields[4]})
	}
	return out, nil
}

func parseShow(raw string, withDiff bool, capBytes int) (CommitDetail, error) {
	sep := strings.IndexByte(raw, 0x1e)
	if sep < 0 {
		return CommitDetail{}, fmt.Errorf("malformed Git show record")
	}
	meta := strings.Split(raw[:sep], string(rune(0x1f)))
	if len(meta) != 6 {
		return CommitDetail{}, fmt.Errorf("malformed Git show metadata")
	}
	date, err := time.Parse(time.RFC3339, strings.TrimSpace(meta[3]))
	if err != nil {
		return CommitDetail{}, fmt.Errorf("malformed Git commit date: %w", err)
	}
	d := CommitDetail{Commit: Commit{Hash: meta[0], ShortHash: meta[1], Author: meta[2], Date: date, Subject: meta[4]}, Body: strings.TrimSpace(meta[5])}
	rest := raw[sep+1:]
	if withDiff {
		if len(rest) > capBytes {
			d.Diff = rest[:capBytes]
			d.DiffTruncated = true
		} else {
			d.Diff = rest
		}
		return d, nil
	}
	for _, line := range strings.Split(rest, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "commit ") || strings.HasPrefix(line, "Author:") || strings.HasPrefix(line, "Date:") || strings.HasPrefix(line, "    ") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && len(parts[0]) <= 3 {
			d.Files = append(d.Files, ChangedFile{Status: parts[0], Path: strings.Join(parts[1:], " ")})
		}
	}
	sort.Slice(d.Files, func(i, j int) bool { return d.Files[i].Path < d.Files[j].Path })
	return d, nil
}
