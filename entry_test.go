// Copyright (c) the go-freedesktop/desktopentry authors
//
// SPDX-License-Identifier: BSD-3-Clause

package desktopentry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const editorPath = "testdata/apps-low/org.example.Editor.desktop"

func TestParseFields(t *testing.T) {
	e, err := ParseFile(editorPath)
	if err != nil {
		t.Fatal(err)
	}
	if e.Type != "Application" {
		t.Errorf("Type = %q", e.Type)
	}
	if e.Name != "Text Editor" {
		t.Errorf("Name = %q", e.Name)
	}
	if e.GenericName != "Editor" {
		t.Errorf("GenericName = %q", e.GenericName)
	}
	if e.Comment != "Edit text files" {
		t.Errorf("Comment = %q", e.Comment)
	}
	if e.Exec != "editor %F" || e.TryExec != "editor" {
		t.Errorf("Exec/TryExec = %q / %q", e.Exec, e.TryExec)
	}
	if e.Icon != "accessories-text-editor" {
		t.Errorf("Icon = %q", e.Icon)
	}
	if e.StartupWMClass != "Editor" {
		t.Errorf("StartupWMClass = %q", e.StartupWMClass)
	}
	if e.Terminal {
		t.Error("Terminal should be false")
	}
	if strings.Join(e.Categories, ",") != "Utility,TextEditor" {
		t.Errorf("Categories = %v", e.Categories)
	}
	if strings.Join(e.Keywords, ",") != "text,editor" {
		t.Errorf("Keywords = %v", e.Keywords)
	}
	if strings.Join(e.MimeType, ",") != "text/plain" {
		t.Errorf("MimeType = %v", e.MimeType)
	}
	if e.Path != editorPath {
		t.Errorf("Path = %q", e.Path)
	}
}

func TestParseActions(t *testing.T) {
	e, err := ParseFile(editorPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Actions) != 2 {
		t.Fatalf("want 2 actions, got %d", len(e.Actions))
	}
	nw := e.Actions[0]
	if nw.Name != "New Window" || nw.Exec != "editor --new-window" || nw.Icon != "window-new" {
		t.Errorf("action[0] = %+v", nw)
	}
	// Expand an action's Exec through the same machinery.
	e.Exec = e.Actions[0].Exec
	argv, err := e.ExpandExec(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(argv) != 2 || argv[0] != "editor" || argv[1] != "--new-window" {
		t.Errorf("action expand = %v", argv)
	}
}

func TestParseLocalizedNames(t *testing.T) {
	if e := mustNames(t); e[""] != "Text Editor" || e["fr"] != "Éditeur de texte" {
		t.Errorf("Names = %v", e)
	}
}

func mustNames(t *testing.T) map[string]string {
	t.Helper()
	e, err := ParseFile(editorPath)
	if err != nil {
		t.Fatal(err)
	}
	return e.Names
}

func TestParseWithLocalePick(t *testing.T) {
	f, err := os.Open(editorPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	e, err := ParseWithLocale(f, "fr_FR.UTF-8")
	if err != nil {
		t.Fatal(err)
	}
	if e.Name != "Éditeur de texte" {
		t.Errorf("localized Name = %q", e.Name)
	}
	if e.GenericName != "Éditeur" {
		t.Errorf("localized GenericName = %q", e.GenericName)
	}
	if strings.Join(e.Keywords, ",") != "texte,éditeur" {
		t.Errorf("localized Keywords = %v", e.Keywords)
	}
}

func TestParseWithLocaleBad(t *testing.T) {
	// keyfile.ParseLocale rejects a locale with an empty language part.
	if _, err := ParseWithLocale(strings.NewReader("[Desktop Entry]\nType=Application\nName=x\n"), "_TERRITORY"); err == nil {
		t.Fatal("want error for malformed locale")
	}
}

func TestParseReadError(t *testing.T) {
	if _, err := Parse(errReader{}); err == nil {
		t.Fatal("want read error")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, os.ErrClosed }

func TestParseInvalidEntry(t *testing.T) {
	// Missing Type -> the underlying parser errors.
	if _, err := Parse(strings.NewReader("[Desktop Entry]\nName=x\n")); err == nil {
		t.Fatal("want error for missing Type")
	}
}

func TestParseFileMissing(t *testing.T) {
	if _, err := ParseFile("testdata/does-not-exist.desktop"); err == nil {
		t.Fatal("want error for missing file")
	}
	// A path that exists but is not a valid entry -> parse error branch.
	if _, err := ParseFile("testdata/apps-low/notes.txt"); err == nil {
		t.Fatal("want error for non-desktop file")
	}
}

func TestNamesNoNameKey(t *testing.T) {
	// A Directory entry has no Name requirement? Actually Name is required;
	// use a keyfile with no Name-family keys reached via localizedNames on a
	// group that parses but whose only keys are non-Name.
	if n := localizedNames([]byte("[Desktop Entry]\nType=Application\nName=x\nExec=y\n")); n[""] != "x" {
		t.Errorf("names = %v", n)
	}
	// Unparsable keyfile -> nil map.
	if n := localizedNames([]byte("no group here")); n != nil {
		t.Errorf("want nil, got %v", n)
	}
	// Group with keys but none in the Name family -> nil map.
	if n := localizedNames([]byte("[Other]\nFoo=bar\n")); n != nil {
		t.Errorf("want nil for no Name keys, got %v", n)
	}
}

func TestShouldShowIn(t *testing.T) {
	e := &Entry{OnlyShowIn: []string{"GNOME"}}
	if !e.ShouldShowIn("") {
		t.Error("empty current should always show")
	}
	if !e.ShouldShowIn("GNOME") {
		t.Error("GNOME should show")
	}
	if e.ShouldShowIn("KDE") {
		t.Error("KDE should be hidden by OnlyShowIn")
	}

	e2 := &Entry{NotShowIn: []string{"KDE"}}
	if e2.ShouldShowIn("KDE") {
		t.Error("KDE should be hidden by NotShowIn")
	}
	if !e2.ShouldShowIn("GNOME") {
		t.Error("GNOME should show when only NotShowIn=KDE")
	}
}

func TestParseFileClosesAndAbsPath(t *testing.T) {
	// Sanity: Path is exactly what we passed (may be relative).
	p := filepath.Clean(editorPath)
	e, err := ParseFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if e.Path != p {
		t.Errorf("Path = %q, want %q", e.Path, p)
	}
}
