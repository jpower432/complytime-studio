// SPDX-License-Identifier: Apache-2.0

package auth

import "os"

// APITokenFromEnv returns the STUDIO_API_TOKEN value. Empty means disabled.
func APITokenFromEnv() string {
	return os.Getenv("STUDIO_API_TOKEN")
}
