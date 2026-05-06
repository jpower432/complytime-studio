// SPDX-License-Identifier: Apache-2.0

const { defineConfig } = require("cypress");

module.exports = defineConfig({
  e2e: {
    supportFile: false,
    baseUrl: process.env.CYPRESS_BASE_URL || "http://127.0.0.1:5173",
    video: true,
    screenshotOnRunFailure: true,
    defaultCommandTimeout: 15000,
  },
});
