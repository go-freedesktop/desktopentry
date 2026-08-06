// Copyright (c) the go-freedesktop/desktopentry authors
//
// SPDX-License-Identifier: BSD-3-Clause

package desktopentry

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adrg/xdg"
)

// Scan enumerates every installed application as a launcher would: it walks
// all applications/ directories resolved by github.com/adrg/xdg, parses each
// *.desktop file, de-duplicates by desktop-file id (a higher-precedence
// directory wins) and drops the entries a launcher must not show (NoDisplay
// or Hidden). The result is the dock / Spotlight index.
func Scan() []*Entry {
	// xdg.ApplicationDirs is ordered highest-precedence first; ScanDirs
	// treats its argument as increasing precedence (later overrides
	// earlier), so hand it the reversed list.
	return ScanDirs(reversed(xdg.ApplicationDirs))
}

// ScanDirs is the directory-injectable form of [Scan]. The dirs are given in
// increasing order of precedence: when the same desktop-file id appears in
// several directories, the entry from a later directory overrides the one
// from an earlier directory (including a later Hidden tombstone, which
// removes it from the result). Missing directories are ignored.
func ScanDirs(dirs []string) []*Entry {
	byID := map[string]*Entry{}
	for _, dir := range dirs {
		for id, e := range parseDir(dir) {
			byID[id] = e
		}
	}

	out := make([]*Entry, 0, len(byID))
	for _, e := range byID {
		if e.NoDisplay || e.Hidden {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// parseDir parses every *.desktop file under dir (recursively) into a map
// keyed by desktop-file id. The id is the path relative to dir with the
// ".desktop" suffix removed and separators turned into '-'. Files that fail
// to parse are skipped.
func parseDir(dir string) map[string]*Entry {
	out := map[string]*Entry{}
	root := os.DirFS(dir)
	_ = fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(p, ".desktop") {
			return nil
		}
		e, perr := ParseFile(filepath.Join(dir, p))
		if perr != nil {
			return nil
		}
		id := strings.TrimSuffix(p, ".desktop")
		id = strings.ReplaceAll(id, "/", "-")
		e.ID = id
		out[id] = e
		return nil
	})
	return out
}

// Autostart lists the autostart entries for the current desktop environment
// (read from XDG_CURRENT_DESKTOP), honoring Hidden and OnlyShowIn/NotShowIn.
func Autostart() []*Entry {
	return AutostartDirs(autostartDirs(), os.Getenv("XDG_CURRENT_DESKTOP"))
}

// autostartDirs returns the autostart search path (config home first, then
// the system config dirs), highest precedence first.
func autostartDirs() []string {
	dirs := []string{filepath.Join(xdg.ConfigHome, "autostart")}
	for _, d := range xdg.ConfigDirs {
		dirs = append(dirs, filepath.Join(d, "autostart"))
	}
	return dirs
}

// AutostartDirs is the injectable form of [Autostart]. dirs are given in
// decreasing order of precedence (config home first); an entry present in an
// earlier directory shadows one with the same filename in a later directory.
// Entries that are Hidden or not visible in current (per OnlyShowIn /
// NotShowIn) are dropped.
func AutostartDirs(dirs []string, current string) []*Entry {
	seen := map[string]bool{}
	var out []*Entry
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, de := range entries {
			name := de.Name()
			if de.IsDir() || !strings.HasSuffix(name, ".desktop") {
				continue
			}
			if seen[name] {
				continue
			}
			seen[name] = true
			e, perr := ParseFile(filepath.Join(dir, name))
			if perr != nil {
				continue
			}
			if e.Hidden || !e.ShouldShowIn(current) {
				continue
			}
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func reversed(in []string) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[len(in)-1-i] = v
	}
	return out
}
