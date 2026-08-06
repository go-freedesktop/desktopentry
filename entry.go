// Copyright (c) the go-freedesktop/desktopentry authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package desktopentry is the freedesktop Desktop Entry layer for a
// launcher: it enumerates the installed applications and expands their
// launch commands.
//
// It does not reinvent the low-level parsers. The base .desktop / keyfile
// parse is delegated to github.com/rkoesters/xdg/desktop and the XDG
// base-directory resolution to github.com/adrg/xdg. On top of those this
// package adds the launcher-grade gap:
//
//   - a clean [Entry] type exposing exactly the fields a dock / launcher /
//     Spotlight index needs, with per-locale names;
//   - [Entry.Actions], the [Desktop Action <name>] groups;
//   - [Entry.ExpandExec] / [ExpandExec], correct Exec field-code expansion
//     (%f %F %u %U %i %c %k %%) per the Desktop Entry Specification;
//   - [Scan], which enumerates every applications/ directory, de-duplicates
//     by desktop-file id and drops the entries a launcher must not show;
//   - [Autostart], the autostart/ entries honoring Hidden / OnlyShowIn.
package desktopentry

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/rkoesters/xdg/desktop"
	"github.com/rkoesters/xdg/keyfile"
)

// group / key names used when re-reading raw localized values.
const (
	groupDesktopEntry = "Desktop Entry"
	keyName           = "Name"
)

// Errors returned by this package.
var (
	// ErrNoExec is returned by [ExpandExec] when the entry has no Exec key
	// (for example a Link or Directory entry, which cannot be launched).
	ErrNoExec = errors.New("desktopentry: entry has no Exec")

	// ErrBadExec is returned when the Exec value cannot be tokenized, e.g.
	// it contains an unterminated double quote.
	ErrBadExec = errors.New("desktopentry: malformed Exec value")
)

// Action is a single [Desktop Action <name>] group: an extra launch verb a
// launcher can offer next to the application (for example "New Window").
type Action struct {
	// ID is the action identifier as listed in the Actions key.
	ID string
	// Name is the (localized) user-visible label.
	Name string
	// Exec is the program to run for this action; expand it with
	// [ExpandExec].
	Exec string
	// Icon is an optional icon name / path for the action.
	Icon string
}

// Entry is a parsed desktop entry reduced to the fields a launcher needs.
type Entry struct {
	// ID is the desktop-file id (path under an applications/ dir with the
	// .desktop suffix stripped and directory separators turned into '-',
	// e.g. "kde4-konsole"). It is empty for entries parsed directly with
	// [Parse] rather than discovered by [Scan].
	ID string
	// Path is the absolute path of the .desktop file, or empty when the
	// entry was parsed from an in-memory reader.
	Path string

	// Type is the raw Type value ("Application", "Link", "Directory", ...).
	Type string

	// Name is the localized application name (resolved for the locale used
	// at parse time).
	Name string
	// Names maps a locale key ("" for the default, otherwise "fr",
	// "sr_Latn", ...) to the raw Name value, for cross-locale search.
	Names map[string]string
	// GenericName is the localized generic name ("Web Browser").
	GenericName string
	// Comment is the localized tooltip / description.
	Comment string

	// Exec is the program command line with field codes; expand it with
	// [Entry.ExpandExec].
	Exec string
	// TryExec is a binary whose presence gates whether the entry is
	// installed.
	TryExec string

	// Icon is the icon name or absolute path (feeds an icon-theme lookup).
	Icon string

	// Categories are the menu categories.
	Categories []string
	// Keywords are the localized search keywords.
	Keywords []string
	// MimeType are the MIME types the application can open.
	MimeType []string

	// NoDisplay hides the entry from menus / launchers.
	NoDisplay bool
	// Hidden marks the entry as deleted (a tombstone); a launcher must
	// treat it as absent.
	Hidden bool
	// Terminal requests the program be run inside a terminal.
	Terminal bool

	// OnlyShowIn / NotShowIn gate visibility by desktop environment.
	OnlyShowIn []string
	NotShowIn  []string

	// StartupWMClass is the WM class the launched window will set, used to
	// map a window back to its entry.
	StartupWMClass string

	// Actions are the extra launch verbs.
	Actions []Action
}

// Parse reads a desktop entry from r using the process default locale.
func Parse(r io.Reader) (*Entry, error) {
	return ParseWithLocale(r, "")
}

// ParseWithLocale reads a desktop entry from r, resolving localized strings
// (Name, GenericName, Comment, Keywords) for locale, e.g. "fr_FR.UTF-8" or
// "sr@latin". An empty locale selects the process default locale.
func ParseWithLocale(r io.Reader, locale string) (*Entry, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	loc := keyfile.DefaultLocale()
	if locale != "" {
		loc, err = keyfile.ParseLocale(locale)
		if err != nil {
			return nil, err
		}
	}

	de, err := desktop.NewWithLocale(bytes.NewReader(raw), loc)
	if err != nil {
		return nil, err
	}

	e := fromDesktop(de)
	e.Names = localizedNames(raw)
	return e, nil
}

// ParseFile reads and parses the desktop entry at path using the default
// locale, recording its Path.
func ParseFile(path string) (*Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	e, err := Parse(f)
	if err != nil {
		return nil, err
	}
	e.Path = path
	return e, nil
}

// fromDesktop copies the rkoesters entry into our launcher-facing type.
func fromDesktop(de *desktop.Entry) *Entry {
	e := &Entry{
		Type:           de.Type.String(),
		Name:           de.Name,
		GenericName:    de.GenericName,
		Comment:        de.Comment,
		Exec:           de.Exec,
		TryExec:        de.TryExec,
		Icon:           de.Icon,
		Categories:     de.Categories,
		Keywords:       de.Keywords,
		MimeType:       de.MimeType,
		NoDisplay:      de.NoDisplay,
		Hidden:         de.Hidden,
		Terminal:       de.Terminal,
		OnlyShowIn:     de.OnlyShowIn,
		NotShowIn:      de.NotShowIn,
		StartupWMClass: de.StartupWMClass,
	}
	for _, a := range de.Actions {
		e.Actions = append(e.Actions, Action{
			Name: a.Name,
			Exec: a.Exec,
			Icon: a.Icon,
		})
	}
	return e
}

// localizedNames re-reads the raw file to collect every Name / Name[locale]
// value, so a launcher can search across languages.
func localizedNames(raw []byte) map[string]string {
	kf, err := keyfile.New(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	names := map[string]string{}
	for _, k := range kf.Keys(groupDesktopEntry) {
		switch {
		case k == keyName:
			names[""] = kf.Value(groupDesktopEntry, k)
		case strings.HasPrefix(k, keyName+"[") && strings.HasSuffix(k, "]"):
			loc := k[len(keyName)+1 : len(k)-1]
			names[loc] = kf.Value(groupDesktopEntry, k)
		}
	}
	if len(names) == 0 {
		return nil
	}
	return names
}

// ShouldShowIn reports whether the entry should be shown in the desktop
// environment named current (as in XDG_CURRENT_DESKTOP), honoring the
// OnlyShowIn and NotShowIn keys. An empty current means "any environment",
// which ignores both keys.
func (e *Entry) ShouldShowIn(current string) bool {
	if current == "" {
		return true
	}
	if len(e.OnlyShowIn) > 0 && !contains(e.OnlyShowIn, current) {
		return false
	}
	if contains(e.NotShowIn, current) {
		return false
	}
	return true
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
