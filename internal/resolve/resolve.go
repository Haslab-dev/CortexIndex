// Package resolve provides safe, bounded discovery for the Cortex executable
// used by agent instructions when it is not already on PATH.
package resolve

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Find resolves an executable using explicit override, PATH, bounded repo
// candidates, user-local install, and GOPATH/bin.
func Find(explicit, cwd, home, pathEnv, goBin string) (string, error) {
	candidates := []string{}
	if explicit != "" {
		candidates = append(candidates, explicit)
	}
	if pathEnv != "" {
		if p, err := exec.LookPath("cortex"); err == nil {
			candidates = append(candidates, p)
		}
	}
	for dir := cwd; dir != ""; dir = filepath.Dir(dir) {
		candidates = append(candidates, filepath.Join(dir, "dist", "cortex"), filepath.Join(dir, "cortex"))
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if home != "" {
		candidates = append(candidates, filepath.Join(home, ".local", "bin", "cortex"))
	}
	if goBin != "" {
		candidates = append(candidates, filepath.Join(goBin, "cortex"))
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if !filepath.IsAbs(candidate) {
			if abs, err := filepath.Abs(candidate); err == nil {
				candidate = abs
			}
		}
		candidate = filepath.Clean(candidate)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("cortex executable not found; install with `make install` or set CORTEX_BIN")
}

// ShellSnippet is embedded in the Skill and AGENTS instructions. It is
// POSIX-compatible and uses only bounded, validated candidates.
func ShellSnippet() string {
	return `cortex_cmd() {
  _cortex_bin="${CORTEX_BIN:-}"
  if [ -z "$_cortex_bin" ] || [ ! -x "$_cortex_bin" ]; then
    _cortex_bin="$(command -v cortex 2>/dev/null || true)"
  fi
  if [ -z "$_cortex_bin" ] || [ ! -x "$_cortex_bin" ]; then
    for _cortex_root in "$PWD" "$(dirname "$PWD")" "$(dirname "$(dirname "$PWD")")"; do
      if [ -x "$_cortex_root/dist/cortex" ]; then _cortex_bin="$_cortex_root/dist/cortex"; break; fi
      if [ -x "$_cortex_root/cortex" ]; then _cortex_bin="$_cortex_root/cortex"; break; fi
    done
  fi
  if [ -z "$_cortex_bin" ] && [ -x "${HOME:-}/.local/bin/cortex" ]; then _cortex_bin="${HOME}/.local/bin/cortex"; fi
  if [ -z "$_cortex_bin" ] || [ ! -x "$_cortex_bin" ]; then
    printf '%s\n' 'Cortex is unavailable; continue with normal exploration and install it with make install.' >&2
    return 127
  fi
  "$_cortex_bin" "$@"
}`
}
