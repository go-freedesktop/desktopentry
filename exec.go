// Copyright (c) the go-freedesktop/desktopentry authors
//
// SPDX-License-Identifier: BSD-3-Clause

package desktopentry

import "strings"

// ExpandExec expands the entry's Exec value into an argv slice ready to be
// handed to os/exec, substituting the Desktop Entry Specification field
// codes:
//
//	%f  a single file path        (the first element of files)
//	%F  the list of file paths    (one argv element each)
//	%u  a single URL              (url, or the first file when url is empty)
//	%U  the list of URLs          ([url] when set, otherwise the files)
//	%i  --icon <Icon>             (two argv elements; nothing if Icon is "")
//	%c  the localized Name
//	%k  the desktop-file path     (Entry.Path)
//	%%  a literal percent sign
//
// The deprecated field codes %d %D %n %N %v %m are recognized and dropped.
// A field code that expands to nothing (for example %f with no files) does
// not leave behind an empty argument.
//
// The Exec value is tokenized per the specification's quoting rules (double
// quotes with backslash escapes) before substitution. An entry without an
// Exec key yields [ErrNoExec]; a malformed value yields [ErrBadExec].
func (e *Entry) ExpandExec(files []string, url string) ([]string, error) {
	return ExpandExec(e, files, url)
}

// ExpandExec is the package-level form of [Entry.ExpandExec].
func ExpandExec(e *Entry, files []string, url string) ([]string, error) {
	if strings.TrimSpace(e.Exec) == "" {
		return nil, ErrNoExec
	}
	tokens, err := tokenizeExec(e.Exec)
	if err != nil {
		return nil, err
	}
	var argv []string
	for _, tok := range tokens {
		argv = append(argv, expandToken(tok, e, files, url)...)
	}
	return argv, nil
}

// tokenizeExec splits an Exec value into arguments following the Desktop
// Entry Specification: arguments are separated by unquoted whitespace, and a
// double-quoted section may contain backslash escapes for " ` $ and \.
func tokenizeExec(s string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	inToken := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inToken = true
			i++
			for i < len(s) && s[i] != '"' {
				if s[i] == '\\' {
					i++
					if i >= len(s) {
						return nil, ErrBadExec
					}
				}
				cur.WriteByte(s[i])
				i++
			}
			if i >= len(s) {
				return nil, ErrBadExec // unterminated quote
			}
		case c == ' ' || c == '\t':
			if inToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				inToken = false
			}
		default:
			inToken = true
			cur.WriteByte(c)
		}
	}
	if inToken {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}

// expandToken substitutes the field codes in a single already-unquoted
// token, returning zero, one, or several argv elements.
func expandToken(tok string, e *Entry, files []string, url string) []string {
	var out []string
	var cur strings.Builder
	used := false // cur holds real content worth emitting

	flush := func() {
		if used {
			out = append(out, cur.String())
			cur.Reset()
			used = false
		}
	}

	for i := 0; i < len(tok); i++ {
		if tok[i] != '%' || i+1 >= len(tok) {
			cur.WriteByte(tok[i])
			used = true
			continue
		}
		i++
		switch tok[i] {
		case '%':
			cur.WriteByte('%')
			used = true
		case 'f':
			if len(files) > 0 {
				cur.WriteString(files[0])
				used = true
			}
		case 'u':
			if u := singleURL(files, url); u != "" {
				cur.WriteString(u)
				used = true
			}
		case 'c':
			if e.Name != "" {
				cur.WriteString(e.Name)
				used = true
			}
		case 'k':
			if e.Path != "" {
				cur.WriteString(e.Path)
				used = true
			}
		case 'F':
			flush()
			out = append(out, files...)
		case 'U':
			flush()
			if url != "" {
				out = append(out, url)
			} else {
				out = append(out, files...)
			}
		case 'i':
			flush()
			if e.Icon != "" {
				out = append(out, "--icon", e.Icon)
			}
		case 'd', 'D', 'n', 'N', 'v', 'm':
			// deprecated: drop.
		default:
			// unknown code: drop per spec (undefined behavior).
		}
	}
	flush()
	return out
}

func singleURL(files []string, url string) string {
	if url != "" {
		return url
	}
	if len(files) > 0 {
		return files[0]
	}
	return ""
}
