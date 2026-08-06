// Copyright (c) the go-freedesktop/desktopentry authors
//
// SPDX-License-Identifier: BSD-3-Clause

package desktopentry_test

import (
	"fmt"
	"strings"

	"github.com/go-freedesktop/desktopentry"
)

// ExampleExpandExec shows turning a clicked entry plus a file into the exact
// argv a launcher should exec.
func ExampleExpandExec() {
	e, err := desktopentry.Parse(strings.NewReader(
		"[Desktop Entry]\nType=Application\nName=Gedit\nExec=gedit %U\nIcon=gedit\n",
	))
	if err != nil {
		panic(err)
	}

	argv, err := e.ExpandExec(nil, "file:///tmp/notes.txt")
	if err != nil {
		panic(err)
	}
	fmt.Println(argv)
	// Output: [gedit file:///tmp/notes.txt]
}
