package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"cortex/internal/history"
)

func cmdHistory(args []string) error {
	root, err := findRoot(flagOr(args, "--dir"))
	if err != nil {
		return err
	}
	query, err := historyQuery(args)
	if err != nil {
		return err
	}
	commit := flagOr(args, "--commit")
	withDiff := hasFlag(args, "--diff")
	if withDiff && commit == "" {
		return fmt.Errorf("--diff requires --commit")
	}
	if commit != "" && query != "" {
		return fmt.Errorf("a query cannot be combined with --commit")
	}
	limit, err := historyLimit(args)
	if err != nil {
		return err
	}
	client := history.New(root)
	if commit != "" {
		detail, err := client.Show(context.Background(), commit, history.ShowOptions{File: flagOr(args, "--file"), WithDiff: withDiff})
		if err != nil {
			return err
		}
		fmt.Print(formatCommit(detail))
		return nil
	}
	commits, err := client.Log(context.Background(), history.LogOptions{Query: query, File: flagOr(args, "--file"), Since: flagOr(args, "--since"), Limit: limit})
	if err != nil {
		return err
	}
	fmt.Print(formatHistory(root, commits))
	return nil
}

func historyQuery(args []string) (string, error) {
	var query string
	for i, arg := range args {
		if strings.HasPrefix(arg, "--") {
			if arg == "--file" || arg == "--limit" || arg == "--since" || arg == "--commit" || arg == "--dir" {
				if i+1 < len(args) {
					continue
				}
				return "", fmt.Errorf("flag %s requires a value", arg)
			}
			continue
		}
		if i > 0 && (args[i-1] == "--file" || args[i-1] == "--limit" || args[i-1] == "--since" || args[i-1] == "--commit" || args[i-1] == "--dir") {
			continue
		}
		if query != "" {
			return "", fmt.Errorf("history accepts at most one query")
		}
		query = arg
	}
	return query, nil
}

func historyLimit(args []string) (int, error) {
	raw := flagOr(args, "--limit")
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("--limit must be a positive integer")
	}
	if n > history.MaxLimit {
		return 0, fmt.Errorf("--limit must be at most %d", history.MaxLimit)
	}
	return n, nil
}

func formatHistory(root string, commits []history.Commit) string {
	var b strings.Builder
	b.WriteString("# Git History\n\n")
	b.WriteString(fmt.Sprintf("Repository: `%s`\n\n", root))
	if len(commits) == 0 {
		b.WriteString("_No matching commits._\n")
		return b.String()
	}
	for _, c := range commits {
		b.WriteString(fmt.Sprintf("- `%s` %s — %s (%s)\n", c.ShortHash, c.Date.Format("2006-01-02"), c.Subject, c.Author))
	}
	return b.String()
}

func formatCommit(d history.CommitDetail) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Git Commit: %s\n\n", d.ShortHash))
	b.WriteString(fmt.Sprintf("- Full hash: `%s`\n- Author: %s\n- Date: %s\n- Subject: %s\n", d.Hash, d.Author, d.Date.Format("2006-01-02 15:04:05Z07:00"), d.Subject))
	if d.Body != "" {
		b.WriteString("\n## Message\n\n" + d.Body + "\n")
	}
	b.WriteString("\n## Changed Files\n\n")
	if len(d.Files) == 0 {
		b.WriteString("_No changed-file list available._\n")
	} else {
		for _, f := range d.Files {
			b.WriteString(fmt.Sprintf("- `%s` `%s`\n", f.Status, f.Path))
		}
	}
	if d.Diff != "" {
		b.WriteString("\n## Diff\n\n```diff\n" + d.Diff + "\n```\n")
		if d.DiffTruncated {
			b.WriteString("\n> Diff truncated. Use native `git show` for the complete patch.\n")
		}
	}
	return b.String()
}
