// SPDX-License-Identifier: Apache-2.0
// Plain JS config — no local cypress package needed.
// Works with the globally installed Cypress binary.

module.exports = {
  e2e: {
    baseUrl: process.env.STUDIO_URL || "http://localhost:8080",
    specPattern: "cypress/e2e/**/*.cy.ts",
    supportFile: false,
    video: true,
    videoCompression: false,
    trashAssetsBeforeRuns: true,
    viewportWidth: 1920,
    viewportHeight: 1080,
    defaultCommandTimeout: 15000,
    requestTimeout: 15000,
    responseTimeout: 30000,
    animationDistanceThreshold: 5,
    waitForAnimations: true,
  },
};
