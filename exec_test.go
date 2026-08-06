// Copyright (c) the go-freedesktop/desktopentry authors
//
// SPDX-License-Identifier: BSD-3-Clause

package desktopentry

import (
	"errors"
	"reflect"
	"testing"
)

func TestExpandExec(t *testing.T) {
	base := &Entry{
		Name: "My App",
		Icon: "myicon",
		Path: "/apps/my.desktop",
	}
	files := []string{"/a.txt", "/b.txt"}

	cases := []struct {
		name  string
		exec  string
		files []string
		url   string
		want  []string
	}{
		{"plain", "prog", nil, "", []string{"prog"}},
		{"single-file", "prog %f", files, "", []string{"prog", "/a.txt"}},
		{"single-file-none", "prog %f", nil, "", []string{"prog"}},
		{"multi-file", "prog %F", files, "", []string{"prog", "/a.txt", "/b.txt"}},
		{"multi-file-none", "prog %F", nil, "", []string{"prog"}},
		{"single-url", "prog %u", nil, "http://x", []string{"prog", "http://x"}},
		{"single-url-fallback-file", "prog %u", files, "", []string{"prog", "/a.txt"}},
		{"single-url-none", "prog %u", nil, "", []string{"prog"}},
		{"multi-url", "prog %U", nil, "http://x", []string{"prog", "http://x"}},
		{"multi-url-fallback-files", "prog %U", files, "", []string{"prog", "/a.txt", "/b.txt"}},
		{"icon", "prog %i", nil, "", []string{"prog", "--icon", "myicon"}},
		{"caption", "prog %c", nil, "", []string{"prog", "My App"}},
		{"key-path", "prog %k", nil, "", []string{"prog", "/apps/my.desktop"}},
		{"percent", "prog %%s", nil, "", []string{"prog", "%s"}},
		{"deprecated-dropped", "prog %d %D %n %N %v %m", nil, "", []string{"prog"}},
		{"unknown-dropped", "prog %z", nil, "", []string{"prog"}},
		{"trailing-percent", "prog %", nil, "", []string{"prog", "%"}},
		{"embedded-file", "prog=%f", files, "", []string{"prog=/a.txt"}},
		{"quoted-space", `prog "a b" %f`, files, "", []string{"prog", "a b", "/a.txt"}},
		{"quoted-escape", `prog "a\"b"`, nil, "", []string{"prog", `a"b`}},
		{"tab-separated", "prog\t%f", files, "", []string{"prog", "/a.txt"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := *base
			e.Exec = tc.exec
			got, err := e.ExpandExec(tc.files, tc.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ExpandExec(%q) = %#v, want %#v", tc.exec, got, tc.want)
			}
		})
	}
}

func TestExpandExecIconEmpty(t *testing.T) {
	e := &Entry{Exec: "prog %i", Icon: ""}
	got, err := e.ExpandExec(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"prog"}) {
		t.Fatalf("got %#v", got)
	}
}

func TestExpandExecCaptionAndKeyEmpty(t *testing.T) {
	e := &Entry{Exec: "prog %c %k"} // Name and Path both empty
	got, err := e.ExpandExec(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"prog"}) {
		t.Fatalf("got %#v", got)
	}
}

func TestExpandExecNoExec(t *testing.T) {
	if _, err := ExpandExec(&Entry{Exec: ""}, nil, ""); !errors.Is(err, ErrNoExec) {
		t.Fatalf("want ErrNoExec, got %v", err)
	}
	if _, err := ExpandExec(&Entry{Exec: "   "}, nil, ""); !errors.Is(err, ErrNoExec) {
		t.Fatalf("want ErrNoExec for blank, got %v", err)
	}
}

func TestExpandExecMalformed(t *testing.T) {
	// unterminated quote
	if _, err := ExpandExec(&Entry{Exec: `prog "unterminated`}, nil, ""); !errors.Is(err, ErrBadExec) {
		t.Fatalf("want ErrBadExec, got %v", err)
	}
	// trailing backslash inside quote
	if _, err := ExpandExec(&Entry{Exec: `prog "bad\`}, nil, ""); !errors.Is(err, ErrBadExec) {
		t.Fatalf("want ErrBadExec for trailing escape, got %v", err)
	}
}
