// SPDX-License-Identifier: Apache-2.0

import { defineConfig } from "vite";
import preact from "@preact/preset-vite";

export default defineConfig({
  plugins: [preact()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/invoke": "http://localhost:8080",
      "/api": "http://localhost:8080",
      "/.well-known": "http://localhost:8080",
    },
  },
});
