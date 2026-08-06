// Copyright (c) the go-freedesktop/desktopentry authors
//
// SPDX-License-Identifier: BSD-3-Clause

package desktopentry

import (
	"testing"
)

func byID(entries []*Entry) map[string]*Entry {
	m := map[string]*Entry{}
	for _, e := range entries {
		m[e.ID] = e
	}
	return m
}

func TestScanDirsDedupOverrideAndFilter(t *testing.T) {
	// apps-low is lower precedence, apps-high overrides it.
	entries := ScanDirs([]string{"testdata/apps-low", "testdata/apps-high"})
	m := byID(entries)

	// Override: the high-precedence Editor wins.
	ed, ok := m["org.example.Editor"]
	if !ok {
		t.Fatal("Editor missing")
	}
	if ed.Name != "Text Editor HI" || ed.Exec != "editor-hi %F" {
		t.Errorf("override failed: %+v", ed)
	}

	// Subdir -> '-' id.
	if _, ok := m["kde4-konsole"]; !ok {
		t.Errorf("kde4-konsole missing; ids=%v", keys(m))
	}

	// NoDisplay filtered.
	if _, ok := m["org.example.Hidden"]; ok {
		t.Error("NoDisplay entry should be filtered")
	}

	// Tombstone: visible in low, Hidden=true in high -> removed.
	if _, ok := m["org.example.Doomed"]; ok {
		t.Error("Hidden override should tombstone the entry")
	}

	// broken.desktop (missing Type) is skipped, not fatal.
	if _, ok := m["broken"]; ok {
		t.Error("broken entry should be skipped")
	}

	// Deterministic sort by ID.
	for i := 1; i < len(entries); i++ {
		if entries[i-1].ID > entries[i].ID {
			t.Fatalf("not sorted: %q > %q", entries[i-1].ID, entries[i].ID)
		}
	}
}

func TestScanDirsLowOnly(t *testing.T) {
	// Without the high dir, the low Editor and Doomed are visible.
	m := byID(ScanDirs([]string{"testdata/apps-low"}))
	if m["org.example.Editor"].Name != "Text Editor" {
		t.Errorf("low Editor name = %q", m["org.example.Editor"].Name)
	}
	if _, ok := m["org.example.Doomed"]; !ok {
		t.Error("Doomed should be visible without the tombstone")
	}
}

func TestScanDirsMissingDir(t *testing.T) {
	if got := ScanDirs([]string{"testdata/nope"}); len(got) != 0 {
		t.Errorf("missing dir should yield nothing, got %v", got)
	}
}

func TestScan(t *testing.T) {
	// Exercises the real xdg.ApplicationDirs path. We cannot assert content
	// (host-dependent) but it must not panic and must return a slice.
	_ = Scan()
}

func TestAutostartDirs(t *testing.T) {
	dirs := []string{"testdata/autostart-home", "testdata/autostart-sys"}

	// Under GNOME: gnome-only shows, notshow is hidden.
	got := AutostartDirs(dirs, "GNOME")
	names := map[string]bool{}
	for _, e := range got {
		names[e.Name] = true
	}
	if !names["GNOME Only"] {
		t.Error("GNOME Only should show under GNOME")
	}
	if names["Not In GNOME"] {
		t.Error("Not In GNOME should be hidden under GNOME")
	}
	if names["Hidden Auto"] {
		t.Error("Hidden entry must be dropped")
	}
	if !names["Sys App"] {
		t.Error("Sys App should show")
	}
	// Shadow: home wins over sys (same filename).
	if !names["Shadow HOME"] || names["Shadow SYS"] {
		t.Errorf("shadowing failed: %v", names)
	}
	// broken.desktop skipped.
	if names["Broken Auto"] {
		t.Error("broken autostart entry should be skipped")
	}
}

func TestAutostartDirsOtherDesktop(t *testing.T) {
	dirs := []string{"testdata/autostart-home", "testdata/autostart-sys"}
	got := AutostartDirs(dirs, "KDE")
	names := map[string]bool{}
	for _, e := range got {
		names[e.Name] = true
	}
	// Under KDE: gnome-only hidden, notshow visible.
	if names["GNOME Only"] {
		t.Error("GNOME Only should be hidden under KDE")
	}
	if !names["Not In GNOME"] {
		t.Error("Not In GNOME should show under KDE")
	}
}

func TestAutostartDirsMissing(t *testing.T) {
	if got := AutostartDirs([]string{"testdata/nope-autostart"}, ""); got != nil {
		t.Errorf("missing dir should yield nil, got %v", got)
	}
}

func TestAutostart(t *testing.T) {
	// Exercises the real config-dir path; content is host-dependent.
	_ = Autostart()
}

func TestReversed(t *testing.T) {
	in := []string{"a", "b", "c"}
	out := reversed(in)
	if out[0] != "c" || out[1] != "b" || out[2] != "a" {
		t.Fatalf("reversed = %v", out)
	}
	if in[0] != "a" {
		t.Error("reversed must not mutate input")
	}
}

func keys(m map[string]*Entry) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
