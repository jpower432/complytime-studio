// SPDX-License-Identifier: Apache-2.0

package blob

import "testing"

func TestNormalizeEndpoint(t *testing.T) {
	t.Parallel()
	host, ssl := normalizeEndpoint("minio.example:9000", false)
	if host != "minio.example:9000" || ssl {
		t.Fatalf("got %q %v", host, ssl)
	}
	host, ssl = normalizeEndpoint("https://s3.amazonaws.com", false)
	if host != "s3.amazonaws.com" || !ssl {
		t.Fatalf("https: got %q %v", host, ssl)
	}
	host, ssl = normalizeEndpoint("http://127.0.0.1:9000", true)
	if host != "127.0.0.1:9000" || ssl {
		t.Fatalf("http: got %q %v", host, ssl)
	}
}
