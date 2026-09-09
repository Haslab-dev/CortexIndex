package taste

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cortex/internal/memory"
)

type Manifest struct {
	ID        string
	Name      string
	Version   string
	Provider  string
	Format    int
	UpdatedAt string
	Body      string
}

type Package struct {
	Root     string
	Manifest Manifest
	Records  []memory.Preference
	Digest   string
}

func LoadPackage(root string) (Package, error) {
	manifest, err := parseManifest(filepath.Join(root, "manifest.md"))
	if err != nil {
		return Package{}, err
	}
	if err := validateID(manifest.ID); err != nil {
		return Package{}, err
	}
	if manifest.Format != 1 {
		return Package{}, fmt.Errorf("unsupported Taste package format %d", manifest.Format)
	}
	if _, err := time.Parse("2006-01-02", manifest.UpdatedAt); err != nil {
		return Package{}, fmt.Errorf("manifest updated_at must be YYYY-MM-DD")
	}
	prefDir := filepath.Join(root, "preferences")
	entries, err := os.ReadDir(prefDir)
	if err != nil {
		return Package{}, fmt.Errorf("read preferences: %w", err)
	}
	var records []memory.Preference
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		path := filepath.Join(prefDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return Package{}, err
		}
		record, ok, err := memory.ParsePreference(filepath.ToSlash(filepath.Join("preferences", entry.Name())), string(data))
		if !ok {
			return Package{}, fmt.Errorf("%s is not a preference record", entry.Name())
		}
		if err != nil {
			return Package{}, err
		}
		if seen[record.ID] {
			return Package{}, fmt.Errorf("duplicate preference ID %q", record.ID)
		}
		seen[record.ID] = true
		record.Package = manifest.ID
		record.PackageVersion = manifest.Version
		record.Provider = manifest.Provider
		record.Status = "proposed"
		record.Source = fmt.Sprintf("taste://%s/%s", manifest.ID, record.ID)
		record.File = filepath.ToSlash(filepath.Join("preferences", entry.Name()))
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	p := Package{Root: root, Manifest: manifest, Records: records}
	p.Digest = digest(p)
	return p, nil
}

func parseManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	block, body, ok := memoryFrontmatter(string(data))
	if !ok || !strings.Contains(block, "cortex: taste-package") {
		return Manifest{}, fmt.Errorf("invalid Taste package manifest")
	}
	m := Manifest{Body: strings.TrimSpace(body)}
	for _, raw := range strings.Split(block, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || line == "cortex: taste-package" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return Manifest{}, fmt.Errorf("malformed manifest line")
		}
		key, value := strings.TrimSpace(parts[0]), strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		switch key {
		case "id":
			m.ID = value
		case "name":
			m.Name = value
		case "version":
			m.Version = value
		case "provider":
			m.Provider = value
		case "updated_at":
			m.UpdatedAt = value
		case "format":
			if _, err := fmt.Sscanf(value, "%d", &m.Format); err != nil {
				return Manifest{}, err
			}
		}
	}
	if m.Name == "" {
		m.Name = m.ID
	}
	if m.Version == "" {
		m.Version = "1.0.0"
	}
	if m.Provider == "" {
		m.Provider = "taste"
	}
	return m, nil
}

func memoryFrontmatter(content string) (string, string, bool) {
	if !strings.HasPrefix(content, "---\n") {
		return "", content, false
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return "", content, false
	}
	end += 4
	return content[4:end], content[end+4:], true
}

func validateID(id string) error {
	if id == "" || strings.ContainsAny(id, "/\\ \t\r\n") || id == "." || id == ".." {
		return fmt.Errorf("unsafe Taste package ID %q", id)
	}
	return nil
}

func digest(p Package) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%d\x00%s\x00%s\x00%s\x00%s\n", p.Manifest.Format, p.Manifest.ID, p.Manifest.Version, p.Manifest.Provider, p.Manifest.UpdatedAt))
	for _, r := range p.Records {
		b.WriteString(r.ID + "\x00" + r.Key + "\x00" + r.Scope + "\x00" + r.Text + "\x00" + fmt.Sprintf("%.6f", r.Confidence) + "\n")
	}
	sum := sha256.Sum256([]byte(b.String()))
	return "sha256:" + hex.EncodeToString(sum[:])
}
