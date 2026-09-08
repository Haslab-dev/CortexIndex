package main

import (
	"fmt"
	"strconv"

	"cortex/internal/context"
)

func cmdOverview(args []string) error {
	root, err := findRoot(flagOr(args, "--dir"))
	if err != nil {
		return err
	}
	st, err := openStore(root)
	if err != nil {
		return err
	}
	defer st.Close()
	budget := context.DefaultBudget
	if v, ok := flagValue(args, "--budget"); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			budget = n
		}
	}
	fmt.Print(context.Build("repository architecture overview", context.Options{
		Root: root, Store: st, HasIndex: indexExists(st), Budget: budget,
		Intent: context.IntentOverview, ExplicitIntent: true,
	}))
	return nil
}
