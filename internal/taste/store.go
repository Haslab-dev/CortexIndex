package taste

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cortex/internal/memory"
)

type ImportOptions struct {
	Enable  bool
	Replace bool
}

func ProjectDir(root string) string { return filepath.Join(root, ".cortex", "taste", "packages") }
func GlobalDir(home string) string  { return filepath.Join(home, ".cortex", "taste", "packages") }

func Import(root, packagePath string, opts ImportOptions) (Package, error) {
	pkg, err := LoadPackage(packagePath)
	if err != nil {
		return Package{}, err
	}
	base := ProjectDir(root)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return Package{}, err
	}
	target := filepath.Join(base, pkg.Manifest.ID)
	if _, err := os.Stat(target); err == nil && !opts.Replace {
		return Package{}, fmt.Errorf("Taste package %q already exists; use --replace", pkg.Manifest.ID)
	}
	if err := copyTree(packagePath, target); err != nil {
		return Package{}, err
	}
	prefDir := filepath.Join(root, ".cortex", "preferences")
	if err := os.MkdirAll(prefDir, 0o755); err != nil {
		return Package{}, err
	}
	for _, pref := range pkg.Records {
		if opts.Enable {
			pref.Status = "active"
		}
		path := filepath.Join(prefDir, pkg.Manifest.ID+"--"+pref.ID+".md")
		if err := os.WriteFile(path, []byte(renderPreference(pref)), 0o644); err != nil {
			return Package{}, err
		}
	}
	return pkg, nil
}

func Export(root, packageID, destination string) error {
	if err := validateID(packageID); err != nil {
		return err
	}
	src := filepath.Join(ProjectDir(root), packageID)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("Taste package %q not found", packageID)
	}
	return copyTree(src, filepath.Join(destination, packageID))
}

func List(root string) ([]Package, error) {
	entries, err := os.ReadDir(ProjectDir(root))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Package
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := LoadPackage(filepath.Join(ProjectDir(root), e.Name()))
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest.ID < out[j].Manifest.ID })
	return out, nil
}

func SetStatus(root, packageID, prefID, status string) error {
	valid := map[string]bool{"active": true, "disabled": true, "proposed": true, "deprecated": true, "superseded": true}
	if !valid[status] {
		return fmt.Errorf("invalid Taste status %q", status)
	}
	prefs, err := memory.LoadPreferences(root)
	if err != nil {
		return err
	}
	for _, p := range prefs {
		if p.Package == packageID && p.ID == prefID {
			return rewritePreference(root, p, status, p.Confidence)
		}
	}
	return fmt.Errorf("Taste preference %s/%s not found", packageID, prefID)
}

func SetConfidence(root, packageID, prefID, mode string, value float64) error {
	prefs, err := memory.LoadPreferences(root)
	if err != nil {
		return err
	}
	for _, p := range prefs {
		if p.Package == packageID && p.ID == prefID {
			n := value
			if mode == "delta" {
				n = p.Confidence + value
			}
			if n < 0 || n > 1 {
				return fmt.Errorf("confidence must remain between 0 and 1")
			}
			return rewritePreference(root, p, p.Status, n)
		}
	}
	return fmt.Errorf("Taste preference %s/%s not found", packageID, prefID)
}

func rewritePreference(root string, p memory.Preference, status string, confidence float64) error {
	if p.File == "" {
		return fmt.Errorf("preference has no local file")
	}
	path := filepath.Join(root, ".cortex", filepath.FromSlash(p.File))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(data)
	text = replaceFrontmatterValue(text, "status", status)
	text = replaceFrontmatterValue(text, "confidence", fmt.Sprintf("%.6f", confidence))
	tmp, err := os.CreateTemp(filepath.Dir(path), ".preference-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.WriteString(text); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func replaceFrontmatterValue(content, key, value string) string {
	block, _, ok := memoryFrontmatter(content)
	if !ok {
		return content
	}
	old := key + ":"
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), old) {
			lines[i] = key + ": " + value
		}
	}
	newBlock := strings.Join(lines, "\n")
	end := strings.Index(content[4:], "\n---") + 4
	return "---\n" + newBlock + content[end:]
}

func renderPreference(p memory.Preference) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("cortex: preference\nid: " + p.ID + "\nkey: " + p.Key + "\nscope: " + p.Scope + "\nstatus: " + p.Status + "\n")
	b.WriteString(fmt.Sprintf("confidence: %.6f\nprovider: %s\npackage: %s\npackage_version: %s\nupdated_at: %s\nsource: %s\n---\n\n%s\n", p.Confidence, p.Provider, p.Package, p.PackageVersion, p.UpdatedAt, p.Source, p.Text))
	return b.String()
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}
