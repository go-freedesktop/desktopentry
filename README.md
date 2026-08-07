# desktopentry — go-freedesktop

[![ci](https://github.com/go-freedesktop/desktopentry/actions/workflows/ci.yml/badge.svg)](https://github.com/go-freedesktop/desktopentry/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-freedesktop/desktopentry.svg)](https://pkg.go.dev/github.com/go-freedesktop/desktopentry)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

The freedesktop **[Desktop Entry](https://specifications.freedesktop.org/desktop-entry-spec/latest/)**
layer for a launcher — the piece a dock, application menu, or Spotlight-style
finder needs: **enumerate the installed applications and expand their launch
commands.** Pure Go, **CGO-free**, no runtime dependencies beyond two small
libraries it deliberately reuses.

## Scope — what this adds, and what it reuses

This module does **not** reinvent the low-level parsers. It stands on:

- **[`github.com/rkoesters/xdg`](https://github.com/rkoesters/xdg)** (BSD-3) —
  the base `.desktop` / keyfile parse, including localized values and the
  `[Desktop Action …]` groups.
- **[`github.com/adrg/xdg`](https://github.com/adrg/xdg)** (MIT) — XDG
  base-directory resolution (`applications/`, config dirs).

On top of those it builds the **launcher-grade gap**:

- a clean **`Entry`** type exposing exactly the fields a launcher needs
  (Name + per-locale `Names`, GenericName, Comment, Exec, TryExec, Icon,
  Categories, Keywords, MimeType, NoDisplay, Hidden, Terminal, StartupWMClass,
  Actions);
- **`ExpandExec`** — correct Exec **field-code expansion**
  (`%f %F %u %U %i %c %k %%`, deprecated codes dropped) with spec-compliant
  quote/escape tokenizing — the crux of actually launching an app;
- **`Scan`** — walk every `applications/` dir, de-duplicate by desktop-file id
  (subdir → `-`, higher-precedence dir wins, `Hidden` tombstones), drop
  `NoDisplay`/`Hidden` → the dock / finder index;
- **`Autostart`** — the `autostart/` entries honoring `Hidden` and
  `OnlyShowIn`/`NotShowIn`.

## Install

```sh
go get github.com/go-freedesktop/desktopentry
```

## Quickstart

```go
package main

import (
	"fmt"
	"os/exec"

	"github.com/go-freedesktop/desktopentry"
)

func main() {
	// The launcher / Spotlight index: every visible installed app.
	for _, e := range desktopentry.Scan() {
		fmt.Printf("%-30s %s  (icon: %s)\n", e.ID, e.Name, e.Icon)
	}

	// Launch one, opening a file.
	e, _ := desktopentry.ParseFile("/usr/share/applications/org.gnome.gedit.desktop")
	argv, err := e.ExpandExec([]string{"/tmp/notes.txt"}, "")
	if err != nil {
		panic(err)
	}
	_ = exec.Command(argv[0], argv[1:]...).Start()

	// Offer an entry's actions ("New Window", …) — already parsed.
	for _, a := range e.Actions {
		fmt.Println("action:", a.Name, "→", a.Exec)
	}
}
```

## Public API

| Symbol | Purpose |
| --- | --- |
| `Parse(r) (*Entry, error)` | parse a `.desktop` from a reader, default locale |
| `ParseWithLocale(r, locale) (*Entry, error)` | parse resolving localized strings for `locale` |
| `ParseFile(path) (*Entry, error)` | parse a file, recording `Entry.Path` |
| `Entry` | launcher-facing fields + `Names` map + `Actions` |
| `Action` | a `[Desktop Action …]` group (`ID`, `Name`, `Exec`, `Icon`) |
| `(*Entry).ExpandExec(files, url) ([]string, error)` / `ExpandExec(e, files, url)` | expand Exec field codes into argv |
| `(*Entry).ShouldShowIn(current) bool` | honor `OnlyShowIn`/`NotShowIn` |
| `Scan() []*Entry` | index every visible installed application |
| `ScanDirs(dirs) []*Entry` | injectable form (dirs = increasing precedence) |
| `Autostart() []*Entry` | autostart entries for `$XDG_CURRENT_DESKTOP` |
| `AutostartDirs(dirs, current) []*Entry` | injectable form |
| `ErrNoExec`, `ErrBadExec` | Exec-expansion error sentinels |

### Exec field codes

| Code | Expands to |
| --- | --- |
| `%f` | a single file path (first of `files`) |
| `%F` | the list of file paths (one argv element each) |
| `%u` | a single URL (`url`, or the first file when `url` is empty) |
| `%U` | the list of URLs (`[url]` when set, else the files) |
| `%i` | `--icon <Icon>` (two elements; nothing if `Icon` is empty) |
| `%c` | the localized `Name` |
| `%k` | the desktop-file path (`Entry.Path`) |
| `%%` | a literal `%` |
| `%d %D %n %N %v %m` | deprecated — dropped |

## wasmdesk integration

- `Scan()` → the **Spotlight / dock index** (id, Name, per-locale `Names` for
  cross-language search, Categories).
- `ExpandExec()` → turns a clicked entry (+ optional file/URL) into the exact
  **argv to launch**.
- `Icon` → feeds the future **icontheme** lib to resolve a real image.

## Tests & coverage

`CGO_ENABLED=0 go test ./...` — **100% statement coverage**, including every
error branch, driven by fixtures under `testdata/`. CI additionally
cross-builds and runs the suite on the six supported 64-bit targets
(amd64/arm64 natively, riscv64/loong64/ppc64le/s390x under qemu-user).

## License

BSD-3-Clause. Copyright (c) the go-freedesktop/desktopentry authors.

---

> The `go-freedesktop` org [landing page](https://go-freedesktop.github.io/) and
> [MkDocs documentation site](https://go-freedesktop.github.io/docs/) are live.
