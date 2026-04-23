// SPDX-License-Identifier: Apache-2.0

//go:build dev

package workbench

import "testing/fstest"

// Assets is a no-op filesystem for development and CI where dist/ is not built.
var Assets = fstest.MapFS{
	"dist/index.html": &fstest.MapFile{Data: []byte("<!-- dev mode -->")},
}
