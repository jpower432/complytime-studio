// SPDX-License-Identifier: Apache-2.0

//go:build !dev

package workbench

import "embed"

//go:embed dist/*
var Assets embed.FS
