package main

import (
	"fmt"
	"os"
	"strconv"

	"cortex/internal/bench"
)

func cmdBenchmark(args []string) error {
	iterations := 1
	if v, ok := flagValue(args, "--iterations"); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			iterations = n
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	report, err := bench.Run(exe, iterations)
	if err != nil {
		return err
	}
	if hasFlag(args, "--json") {
		data, _ := report.JSON()
		fmt.Println(string(data))
		return nil
	}
	fmt.Print(report.Markdown())
	return nil
}
