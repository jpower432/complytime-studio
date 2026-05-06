// SPDX-License-Identifier: Apache-2.0
/**
 * UX demo: view-consolidation — six-nav shell, dashboard, programs, import overlay,
 * filter-chips views (policies / evidence / inventory), reviews queue.
 *
 * Stubs /auth/me and /api/* so the walkthrough runs against Vite without OAuth or gateway.
 */

function stubStudioApi() {
  const demoUser = {
    login: "demo",
    name: "Demo Admin",
    avatar_url: "",
    email: "demo@example.test",
    role: "admin",
  };

  const demoProgram = {
    id: "demo-program",
    name: "Demo Program",
    guidance_catalog_id: null,
    framework: "SOC 2",
    applicability: [],
    status: "active",
    health: "green",
    description: null,
    metadata: {},
    policy_ids: [],
    environments: [],
    version: 1,
    green_pct: 0,
    red_pct: 0,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  };

  cy.intercept("GET", "/auth/me", { body: demoUser }).as("authMe");
  cy.intercept("GET", "/api/setup-status", {
    body: { needs_setup: false },
  });
  cy.intercept("GET", "/api/config", {
    body: {
      github_org: "complytime",
      github_repo: "complytime-studio",
      model_provider: "",
      model_name: "",
      auto_persist_artifacts: "true",
    },
  });
  cy.intercept("GET", "/api/chat/history", {
    body: { messages: [], taskId: null },
  });

  cy.intercept("GET", "/api/programs", { body: [demoProgram] }).as("programsList");
  cy.intercept("GET", "/api/programs/demo-program", { body: demoProgram }).as(
    "programDetail",
  );
  cy.intercept("GET", "/api/programs/demo-program/recommendations", {
    body: [],
  }).as("programRecs");

  cy.intercept("GET", "/api/policies", { body: [] });
  cy.intercept("GET", "/api/posture", { body: [] });
  cy.intercept("GET", "/api/catalogs?type=GuidanceCatalog", { body: [] });
  cy.intercept("GET", "/api/inventory", { body: [] });
  cy.intercept("GET", "/api/inventory?*", { body: [] });
  cy.intercept("GET", "/api/evidence?*", { body: [] });
  cy.intercept("GET", "/api/certifications?*", { body: [] });

  cy.intercept("GET", "/api/draft-audit-logs?*", { body: [] });
  cy.intercept("GET", "/api/draft-audit-logs/**", { body: {} });

  cy.intercept("GET", "/api/notifications/unread-count", { body: { count: 0 } });
  cy.intercept("GET", "/api/notifications?limit=*", { body: [] });
  cy.intercept("GET", "/api/notifications?unread=true", { body: [] });
}

describe("view-consolidation demo", () => {
  beforeEach(() => {
    stubStudioApi();
  });

  it("walks the consolidated navigation model", () => {
    cy.visit("/#/dashboard");
    cy.get(".app-loading").should("not.exist");
    cy.get("main.app-main").should("have.attr", "data-view", "dashboard");
    cy.get("section.dashboard-view").should("be.visible");
    cy.contains("button.sidebar-item", "Dashboard").should(
      "have.class",
      "active",
    );

    cy.contains("button.sidebar-item", "Programs").click();
    cy.get("main.app-main").should("have.attr", "data-view", "programs");
    cy.get("div.programs-view").should("be.visible");

    cy.contains("button.sidebar-item", "Policies").click();
    cy.get("div.policies-view").should("be.visible");
    cy.get("main.app-main").should("have.attr", "data-view", "policies");

    cy.contains("button.sidebar-item", "Inventory").click();
    cy.get("section.inventory-view-standalone").should("be.visible");
    cy.get("main.app-main").should("have.attr", "data-view", "inventory");

    cy.contains("button.sidebar-item", "Evidence").click();
    cy.get("section.evidence-view").should("be.visible");
    cy.get("main.app-main").should("have.attr", "data-view", "evidence");

    cy.contains("button.sidebar-item", "Reviews").click();
    cy.get("section.reviews-view").should("be.visible");
    cy.get("main.app-main").should("have.attr", "data-view", "reviews");

    cy.visit("/#/programs/demo-program");
    cy.get("div.program-detail-view").should("be.visible");
    cy.get("main.app-main").should("have.attr", "data-view", "program-detail");
    cy.get("main.app-main").should(
      "have.attr",
      "data-program",
      "demo-program",
    );

    cy.contains("button.sidebar-item", "Dashboard").click();
    cy.get("section.dashboard-view").should("be.visible");

    cy.get('[aria-label="Import Gemara artifact"]').click();
    cy.get(".import-overlay").should("be.visible");
    cy.get(".import-modal").should("be.visible");
    cy.contains("button.btn-secondary", "Cancel").click();
    cy.get(".import-overlay").should("not.exist");

    cy.contains("h1.logo", "ComplyTime Studio").click();
    cy.get("section.dashboard-view").should("be.visible");
  });
});
