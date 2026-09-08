package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cortex/internal/store"
)

const generatedMarker = "cortex: generated"
const generatorVersion = "1"

// BootstrapReport is a deterministic, offline discovery result.
type BootstrapReport struct {
	Proposal string
	Created  []string
	Skipped  []string
}

// Bootstrap discovers project/module facts from the index. It never writes
// unless write is true, and write mode only creates missing/template files.
func Bootstrap(root string, st *store.Store, write bool) (BootstrapReport, error) {
	files, err := st.ListFiles(store.QueryParams{Limit: 100000})
	if err != nil {
		return BootstrapReport{}, err
	}
	syms, err := st.ListSymbols(store.QueryParams{Limit: 100000})
	if err != nil {
		return BootstrapReport{}, err
	}
	imports, err := st.AllImports()
	if err != nil {
		return BootstrapReport{}, err
	}
	mods, err := st.ModuleSummaries()
	if err != nil {
		return BootstrapReport{}, err
	}

	stack := detectStack(root, files)
	project := projectProposal(stack, files, mods)
	architecture := architectureProposal(mods)
	conventions := conventionsProposal()
	proposals := map[string]string{
		"project.md":      project,
		"architecture.md": architecture,
		"conventions.md":  conventions,
	}

	for _, mod := range mods {
		if mod.Path == "." || mod.FileCount == 0 {
			continue
		}
		var source []store.File
		for _, f := range files {
			if modulePathFor(f.Path) == mod.Path {
				source = append(source, f)
			}
		}
		var moduleSyms []store.SymbolRow
		for _, s := range syms {
			if modulePathFor(s.File) == mod.Path {
				moduleSyms = append(moduleSyms, s)
			}
		}
		var moduleImports []store.ImportRow
		for _, i := range imports {
			if modulePathFor(i.File) == mod.Path {
				moduleImports = append(moduleImports, i)
			}
		}
		name := moduleFilename(mod.Path)
		proposals[filepath.ToSlash(filepath.Join("modules", name+".md"))] = moduleProposal(mod.Path, source, moduleSyms, moduleImports)
	}

	keys := make([]string, 0, len(proposals))
	for k := range proposals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var report strings.Builder
	report.WriteString("# Cortex Bootstrap Proposal\n\n")
	report.WriteString("Deterministic offline discovery; no model or network was used.\n\n")
	report.WriteString("## Detected Stack\n\n")
	for _, s := range stack {
		report.WriteString("- " + s + "\n")
	}
	report.WriteString("\n## Proposed Memory\n\n")
	for _, k := range keys {
		report.WriteString("### .cortex/" + k + "\n\n")
		report.WriteString(proposals[k])
		report.WriteString("\n\n")
	}

	reportResult := BootstrapReport{Proposal: report.String()}
	if !write {
		return reportResult, nil
	}
	for _, k := range keys {
		path := filepath.Join(root, DirName, filepath.FromSlash(k))
		old, readErr := os.ReadFile(path)
		if readErr == nil && !IsTemplateContent(string(old)) {
			reportResult.Skipped = append(reportResult.Skipped, k)
			continue
		}
		if readErr != nil && !os.IsNotExist(readErr) {
			return reportResult, readErr
		}
		if err := atomicMemoryWrite(path, []byte(proposals[k]), 0o644); err != nil {
			return reportResult, err
		}
		reportResult.Created = append(reportResult.Created, k)
	}
	return reportResult, nil
}

// IsTemplateContent reports whether memory still contains only the scaffold
// guidance. It is intentionally conservative: unknown content is preserved.
func IsTemplateContent(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return true
	}
	if strings.Contains(trimmed, generatedMarker) {
		return false
	}
	content = stripScaffoldComments(trimmed)
	for _, phrase := range []string{
		"What is this project?", "Primary languages, frameworks", "Keep this small",
		"What is this module?", "What this module owns", "Add repository-wide",
		"Feature-based? Layered? Services?", "Rules the codebase follows",
	} {
		content = strings.ReplaceAll(content, phrase, "")
	}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || line == "<!-- -->" {
			continue
		}
		return false
	}
	return true
}

func stripScaffoldComments(s string) string {
	for {
		start := strings.Index(s, "<!--")
		if start < 0 {
			return s
		}
		end := strings.Index(s[start:], "-->")
		if end < 0 {
			return s[:start]
		}
		s = s[:start] + s[start+end+3:]
	}
}

func detectStack(root string, files []store.File) []string {
	seen := map[string]bool{}
	var out []string
	add := func(v string) {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	for _, f := range files {
		switch strings.ToLower(filepath.Ext(f.Path)) {
		case ".go":
			add("Go")
		case ".ts", ".tsx":
			add("TypeScript")
		case ".js", ".jsx":
			add("JavaScript")
		case ".py":
			add("Python")
		case ".rs":
			add("Rust")
		case ".java":
			add("Java")
		case ".kt", ".kts":
			add("Kotlin")
		case ".c", ".h":
			add("C")
		case ".cpp", ".cc", ".hpp":
			add("C++")
		}
	}
	for _, manifest := range []struct{ file, stack string }{
		{"go.mod", "Go modules"}, {"package.json", "Node package"},
		{"Cargo.toml", "Cargo"}, {"pyproject.toml", "Python project"},
	} {
		if _, err := os.Stat(filepath.Join(root, manifest.file)); err == nil {
			add(manifest.stack)
		}
	}
	sort.Strings(out)
	return out
}

func projectProposal(stack []string, files []store.File, mods []store.ModuleFileStats) string {
	var b strings.Builder
	b.WriteString(generatedHeader([]string{"README.md", "go.mod"}))
	b.WriteString("# Project\n\n## Purpose\n\nCortex-managed project overview generated from repository structure. Review and refine this description.\n\n## Stack\n\n")
	for _, s := range stack {
		b.WriteString("- " + s + "\n")
	}
	b.WriteString("\n## Major Modules\n\n")
	for _, m := range mods {
		if m.Path != "." {
			b.WriteString("- `" + m.Path + "` — " + fmt.Sprintf("%d files, %d symbols", m.FileCount, m.SymbolCount) + "\n")
		}
	}
	b.WriteString(fmt.Sprintf("\n## Index Snapshot\n\n- Indexed files: %d\n- Generated at: %s\n", len(files), time.Now().UTC().Format(time.RFC3339)))
	return b.String()
}

func architectureProposal(mods []store.ModuleFileStats) string {
	var b strings.Builder
	b.WriteString(generatedHeader(nil))
	b.WriteString("# Architecture\n\n## Observed Layers\n\nRepository files are parsed into symbols/imports/references, stored in a derived SQLite/FTS5 index, then retrieved as compact Markdown context.\n\n## Observed Modules\n\n")
	for _, m := range mods {
		if m.Path != "." {
			b.WriteString(fmt.Sprintf("- `%s` (%d files, %d symbols)\n", m.Path, m.FileCount, m.SymbolCount))
		}
	}
	b.WriteString("\n## Review Note\n\nThis is deterministic structural discovery, not a complete semantic architecture claim.\n")
	return b.String()
}

func conventionsProposal() string {
	return generatedHeader(nil) + "# Conventions\n\n- Keep durable project knowledge in human-readable Markdown.\n- Treat the SQLite index as derived and rebuildable.\n- Prefer symbols and relationships before reading entire files.\n- Run `cortex update` after source changes.\n- Source code takes precedence over stale generated memory.\n- Review inferred bootstrap facts before relying on them.\n"
}

func moduleProposal(path string, files []store.File, syms []store.SymbolRow, imports []store.ImportRow) string {
	var b strings.Builder
	sources := make([]string, 0, len(files))
	var manifest strings.Builder
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	for _, f := range files {
		sources = append(sources, f.Path)
		manifest.WriteString(f.Path + "\x00" + f.Hash + "\n")
	}
	sum := sha256.Sum256([]byte(manifest.String()))
	b.WriteString(generatedFrontmatter(sources, "sha256:"+hex.EncodeToString(sum[:])))
	b.WriteString("# Module: " + path + "\n\n## Responsibility\n\nObserved source grouping for `" + path + "`. Semantic responsibility requires review.\n\n## Important Symbols\n\n")
	seen := map[string]bool{}
	for _, s := range syms {
		if s.Parent != "" || seen[s.Display()] {
			continue
		}
		seen[s.Display()] = true
		b.WriteString("- `" + s.Display() + "` — " + string(s.Kind) + "\n")
		if len(seen) == 12 {
			break
		}
	}
	b.WriteString("\n## Dependencies\n\n")
	seen = map[string]bool{}
	for _, i := range imports {
		if !seen[i.Path] {
			seen[i.Path] = true
			b.WriteString("- `" + i.Path + "`\n")
		}
	}
	b.WriteString("\n## Verification\n\nFacts above are derived from the current structural index.\n")
	return b.String()
}

func generatedHeader(sources []string) string { return generatedFrontmatter(sources, "") }

func generatedFrontmatter(sources []string, sourceHash string) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("cortex: generated\ngenerator: deterministic\ngenerator_version: \"" + generatorVersion + "\"\n")
	if len(sources) > 0 {
		b.WriteString("source:\n")
		for _, s := range sources {
			b.WriteString("  - " + s + "\n")
		}
	}
	if sourceHash != "" {
		b.WriteString("source_hash: " + sourceHash + "\n")
	}
	b.WriteString("status: active\n---\n\n")
	return b.String()
}

func moduleFilename(path string) string {
	return strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(path)
}

func modulePathFor(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) <= 1 {
		return "."
	}
	if parts[0] == "cmd" || parts[0] == "internal" || parts[0] == "pkg" || parts[0] == "src" {
		return parts[0] + "/" + parts[1]
	}
	return parts[0]
}

func atomicMemoryWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".cortex-memory-")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
