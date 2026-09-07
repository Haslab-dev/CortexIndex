// Command cortex is a local-first codebase memory engine for AI coding
// agents: durable Markdown memory plus a derived structural index, exposed
// as agent-friendly Markdown on stdout (PRD §16).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"cortex/internal/context"
	"cortex/internal/index"
	"cortex/internal/lexsearch"
	"cortex/internal/memory"
	"cortex/internal/retrieve"
	"cortex/internal/store"
)

const version = "0.1.0"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	cmd := args[0]
	rest := args[1:]

	var err error
	switch cmd {
	case "init":
		err = cmdInit()
	case "index":
		err = cmdIndex(rest, false)
	case "update":
		err = cmdIndex(rest, true)
	case "search":
		err = requireQuery(rest, cmdSearch)
	case "symbol":
		err = requireQuery(rest, cmdSymbol)
	case "refs":
		err = requireQuery(rest, cmdRefs)
	case "deps":
		err = requireQuery(rest, cmdDeps)
	case "context":
		err = requireQuery(rest, cmdContext)
	case "memory":
		err = cmdMemory()
	case "watch":
		err = cmdWatch(rest)
	case "skill", "skills":
		err = cmdSkill(rest)
	case "agents":
		err = cmdAgents(rest)
	case "version", "--version", "-v":
		fmt.Printf("cortex %s\n", version)
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`Cortex — codebase memory engine for AI agents

Usage:
  cortex init                    Initialize memory scaffold (.cortex/)
  cortex index [--full]          Build or rebuild the index (default: incremental)
  cortex update                  Incrementally index changed files
  cortex search "<query>"        Search symbols and file contents
  cortex symbol "<name>" [--src] Show a symbol (calls, callers, docs, source)
  cortex refs "<name>"           Find references to a symbol
  cortex deps "<name>"           Show dependencies (callees, callers, imports)
  cortex context "<task>"        Build task-specific context for an agent
  cortex memory                  Print all project memory
  cortex watch [--interval S]    Continuously apply incremental updates
  cortex skill install [flags]   Install Agent Skills for coding agents
  cortex agents init [--dir P]  Create/update root AGENTS.md instructions
  cortex version                 Print version

All output is Markdown. Memory lives in .cortex/*.md (human-editable);
the SQLite index in .cortex/index/ is derived and rebuildable.
`)
}

// ---- repo/location plumbing ----

// findRoot locates the repo root: --dir flag, $CORTEX_ROOT, cwd, or the
// nearest ancestor containing .cortex/.
func findRoot(flagDir string) (string, error) {
	if flagDir != "" {
		abs, err := filepath.Abs(flagDir)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	if env := os.Getenv("CORTEX_ROOT"); env != "" {
		return filepath.Abs(env)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if memory.Exists(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd, nil
		}
		dir = parent
	}
}

// openStore opens the index DB, ensuring the directory exists.
func openStore(root string) (*store.Store, error) {
	dbPath := filepath.Join(root, memory.DirName, "index", "codebase.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}
	return store.Open(dbPath)
}

// indexerOptions builds indexing options from config.md.
func indexerOptions(root string) index.Options {
	cfg := memory.LoadConfig(root)
	return index.Options{Root: root, ExtraIgnores: cfg.Ignore, MaxFileSize: cfg.MaxFileSize}
}

func flagValue(args []string, flag string) (string, bool) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// flagOr returns the value of flag, or "" when absent.
func flagOr(args []string, flag string) string {
	v, _ := flagValue(args, flag)
	return v
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func queryOf(args []string) (string, error) {
	var parts []string
	for _, a := range args {
		if a == "--src" || a == "--full" || a == "--interval" || a == "--budget" || a == "--dir" {
			break
		}
		if a == "--dir" {
			break
		}
		parts = append(parts, a)
	}
	// strip flag values that got captured (--interval N, --budget N, --dir X)
	clean := parts[:0]
	skipNext := false
	for _, p := range parts {
		if skipNext {
			skipNext = false
			continue
		}
		if p == "--interval" || p == "--budget" || p == "--dir" {
			skipNext = true
			continue
		}
		clean = append(clean, p)
	}
	if len(clean) == 0 {
		return "", fmt.Errorf("missing query; usage: cortex <command> \"<query>\"")
	}
	return clean[0], nil
}

type queryCmd func(root string, query string, args []string) error

func requireQuery(args []string, fn queryCmd) error {
	query, err := queryOf(args)
	if err != nil {
		return err
	}
	root, err := findRoot(flagOr(args, "--dir"))
	if err != nil {
		return err
	}
	return fn(root, query, args)
}

// ---- commands ----

func cmdInit() error {
	root, err := findRoot("")
	if err != nil {
		return err
	}
	created, err := memory.Init(root)
	if err != nil {
		return err
	}
	fmt.Printf("# Cortex initialized\n\n")
	fmt.Printf("Memory directory: `%s`\n\n", filepath.Join(root, memory.DirName))
	if len(created) > 0 {
		fmt.Println("Created:")
		for _, f := range created {
			fmt.Printf("- %s\n", f)
		}
	} else {
		fmt.Println("Scaffold already present — nothing overwritten.")
	}
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit `.cortex/project.md` and describe the project.")
	fmt.Println("2. Run `cortex index` to build the structural index.")
	fmt.Println("3. Ask: `cortex context \"<task>\"`.")
	return nil
}

func cmdIndex(args []string, incremental bool) error {
	root, err := findRoot(flagOr(args, "--dir"))
	if err != nil {
		return err
	}
	if !memory.Exists(root) {
		fmt.Println("> No `.cortex/` found; creating scaffold first.")
		if _, err := memory.Init(root); err != nil {
			return err
		}
	}
	st, err := openStore(root)
	if err != nil {
		return err
	}
	defer st.Close()

	full := hasFlag(args, "--full")
	var stats *index.Stats
	if full || !incremental {
		stats, err = index.Full(indexerOptions(root), st)
	} else {
		stats, err = index.Incremental(indexerOptions(root), st)
	}
	if err != nil {
		return err
	}
	kind := "update"
	if full {
		kind = "full"
	}
	fmt.Print(index.FormatStats(kind, stats))
	return nil
}

func cmdMemory() error {
	root, err := findRoot("")
	if err != nil {
		return err
	}
	out, err := memory.RenderAll(root)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

func cmdSearch(root, query string, _ []string) error {
	st, err := openStore(root)
	if err != nil {
		return err
	}
	defer st.Close()
	if !indexExists(st) {
		return fallbackSearch(root, query)
	}
	out, err := retrieve.New(st, root).Search(query, 20)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

// fallbackSearch runs the lexical scan when no index is present (PRD §27).
func fallbackSearch(root, query string) error {
	kws := context.Keywords(query)
	hits, err := lexsearch.Scan(root, kws, 15)
	if err != nil {
		return err
	}
	fmt.Printf("# Lexical Search: %s\n\n> No index available — run `cortex index` for structural results.\n\n", query)
	if len(hits) == 0 {
		fmt.Print("_No matches._\n\n")
		return nil
	}
	for _, h := range hits {
		fmt.Printf("- `%s:%d` — %s\n", h.Path, h.Line, h.Text)
	}
	return nil
}

func indexExists(st *store.Store) bool {
	files, _, _, _, err := st.Stats()
	return err == nil && files > 0
}

func cmdSymbol(root, name string, args []string) error {
	st, err := openStore(root)
	if err != nil {
		return err
	}
	defer st.Close()
	out, err := retrieve.New(st, root).Symbol(name, hasFlag(args, "--src"))
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

func cmdRefs(root, name string, _ []string) error {
	st, err := openStore(root)
	if err != nil {
		return err
	}
	defer st.Close()
	out, err := retrieve.New(st, root).Refs(name)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

func cmdDeps(root, name string, _ []string) error {
	st, err := openStore(root)
	if err != nil {
		return err
	}
	defer st.Close()
	out, err := retrieve.New(st, root).Deps(name)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

func cmdContext(root, task string, args []string) error {
	budget := context.DefaultBudget
	if v, ok := flagValue(args, "--budget"); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			budget = n
		}
	}
	st, err := openStore(root)
	if err != nil {
		return err
	}
	defer st.Close()
	out := context.Build(task, context.Options{
		Root:     root,
		Budget:   budget,
		HasIndex: indexExists(st),
		Store:    st,
	})
	fmt.Print(out)
	return nil
}

func cmdWatch(args []string) error {
	interval := 2 * time.Second
	if v, ok := flagValue(args, "--interval"); ok {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		} else if n, err := strconv.Atoi(v); err == nil && n > 0 {
			interval = time.Duration(n) * time.Second
		}
	}
	root, err := findRoot(flagOr(args, "--dir"))
	if err != nil {
		return err
	}
	if !memory.Exists(root) {
		return fmt.Errorf("no .cortex/ in %s — run `cortex init` and `cortex index` first", root)
	}
	st, err := openStore(root)
	if err != nil {
		return err
	}
	defer st.Close()

	fmt.Printf("# Watching %s (every %s)\n\nCtrl-C to stop.\n", root, interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		stats, err := index.Incremental(indexerOptions(root), st)
		if err != nil {
			fmt.Fprintf(os.Stderr, "update error: %v\n", err)
			continue
		}
		if stats.Indexed > 0 || stats.Deleted > 0 {
			fmt.Printf("- %s indexed=%d removed=%d\n",
				time.Now().Format("15:04:05"), stats.Indexed, stats.Deleted)
		}
	}
	return nil
}
