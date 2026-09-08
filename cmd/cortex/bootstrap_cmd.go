package main

import (
	"fmt"

	"cortex/internal/index"
	"cortex/internal/memory"
)

func cmdBootstrap(args []string) error {
	root, err := findRoot(flagOr(args, "--dir"))
	if err != nil {
		return err
	}
	if !memory.Exists(root) {
		if _, err := memory.Init(root); err != nil {
			return err
		}
	}
	st, err := openStore(root)
	if err != nil {
		return err
	}
	defer st.Close()
	if !indexExists(st) {
		if _, err := index.Incremental(indexerOptions(root), st); err != nil {
			return err
		}
	}
	report, err := memory.Bootstrap(root, st, hasFlag(args, "--write"))
	if err != nil {
		return err
	}
	if hasFlag(args, "--write") {
		fmt.Printf("# Cortex Bootstrap\n\nCreated: %d\nSkipped existing/manual: %d\n\n", len(report.Created), len(report.Skipped))
		for _, p := range report.Created {
			fmt.Printf("- created `.cortex/%s`\n", p)
		}
		for _, p := range report.Skipped {
			fmt.Printf("- skipped `.cortex/%s`\n", p)
		}
	} else {
		fmt.Print(report.Proposal)
		fmt.Print("\n> Proposal only. Re-run with `--write` to create missing generated memory.\n")
	}
	return nil
}
