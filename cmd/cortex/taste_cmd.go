package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cortex/internal/feedback"
	"cortex/internal/memory"
	"cortex/internal/taste"
)

func cmdTaste(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cortex taste <import|list|lint|show|enable|disable|confidence|feedback>")
	}
	sub := args[0]
	rest := args[1:]
	root, err := findRoot(flagOr(rest, "--dir"))
	if err != nil {
		return err
	}
	switch sub {
	case "import":
		if len(rest) == 0 {
			return fmt.Errorf("taste import requires a package path")
		}
		pkg, err := taste.Import(root, rest[0], taste.ImportOptions{Enable: hasFlag(rest, "--enable"), Replace: hasFlag(rest, "--replace")})
		if err != nil {
			return err
		}
		fmt.Printf("# Taste Import\n\n- Package: `%s`\n- Version: `%s`\n- Preferences: %d\n- Digest: `%s`\n", pkg.Manifest.ID, pkg.Manifest.Version, len(pkg.Records), pkg.Digest)
	case "list":
		pkgs, err := taste.List(root)
		if err != nil {
			return err
		}
		fmt.Print("# Taste Packages\n\n")
		for _, p := range pkgs {
			fmt.Printf("- `%s` %s (%d preferences, %s)\n", p.Manifest.ID, p.Manifest.Version, len(p.Records), p.Digest)
		}
	case "lint":
		path := ""
		if len(rest) > 0 && !strings.HasPrefix(rest[0], "--") {
			path = rest[0]
		} else {
			return fmt.Errorf("taste lint requires a package path")
		}
		p, err := taste.LoadPackage(path)
		if err != nil {
			return err
		}
		fmt.Printf("# Taste Lint\n\n- `%s` valid (%d preferences, %s)\n", p.Manifest.ID, len(p.Records), p.Digest)
	case "show":
		if len(rest) == 0 {
			return fmt.Errorf("taste show requires a package ID")
		}
		pkgs, err := taste.List(root)
		if err != nil {
			return err
		}
		for _, p := range pkgs {
			if p.Manifest.ID == rest[0] {
				fmt.Printf("# Taste Package: %s\n\n- Version: %s\n- Provider: %s\n- Digest: %s\n\n", p.Manifest.ID, p.Manifest.Version, p.Manifest.Provider, p.Digest)
				for _, r := range p.Records {
					fmt.Print(memory.FormatPreference(r))
				}
				return nil
			}
		}
		return fmt.Errorf("Taste package %q not found", rest[0])
	case "enable", "disable":
		if len(rest) < 2 {
			return fmt.Errorf("taste %s requires PACKAGE and PREFERENCE", sub)
		}
		status := "active"
		if sub == "disable" {
			status = "disabled"
		}
		return taste.SetStatus(root, rest[0], rest[1], status)
	case "confidence":
		if len(rest) < 2 {
			return fmt.Errorf("taste confidence requires PACKAGE and PREFERENCE")
		}
		if raw := flagOr(rest, "--set"); raw != "" {
			n, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return err
			}
			return taste.SetConfidence(root, rest[0], rest[1], "set", n)
		}
		if raw := flagOr(rest, "--delta"); raw != "" {
			n, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return err
			}
			return taste.SetConfidence(root, rest[0], rest[1], "delta", n)
		}
		return fmt.Errorf("taste confidence requires --set or --delta")
	case "feedback":
		if len(rest) < 2 {
			return fmt.Errorf("taste feedback requires TARGET and KIND")
		}
		kind := feedback.Kind(rest[1])
		e := feedback.Event{ID: fmt.Sprintf("manual-%d", time.Now().UnixNano()), Provider: "manual", Kind: kind, Target: rest[0], Timestamp: time.Now().UTC().Format(time.RFC3339), Note: flagOr(rest, "--note")}
		if err := feedback.Append(root, e); err != nil {
			return err
		}
		fmt.Print(feedback.Format([]feedback.Event{e}))
	case "feedback-json":
		var e feedback.Event
		if err := json.NewDecoder(os.Stdin).Decode(&e); err != nil {
			return err
		}
		return feedback.Append(root, e)
	case "feedback-list":
		es, err := feedback.List(root, flagOr(rest, "--kind"), 50)
		if err != nil {
			return err
		}
		fmt.Print(feedback.Format(es))
	case "feedback-learn":
		return fmt.Errorf("automatic learning is review-only; use explicit preference status changes")
	case "push":
		if len(rest) < 2 {
			return fmt.Errorf("taste push requires PACKAGE and DESTINATION")
		}
		return taste.Export(root, rest[0], rest[1])
	case "pull":
		if len(rest) < 1 {
			return fmt.Errorf("taste pull requires PACKAGE PATH")
		}
		_, err := taste.Import(root, rest[1], taste.ImportOptions{})
		return err
	default:
		return fmt.Errorf("unknown taste command %q", sub)
	}
	return nil
}
