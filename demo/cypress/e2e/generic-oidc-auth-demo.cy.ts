// SPDX-License-Identifier: Apache-2.0
//
// Demo: Generic OIDC Authentication
//
// Shows Studio configured with a non-Google OIDC provider (Keycloak),
// demonstrating the provider-agnostic login flow, first-admin promotion,
// and JWT role seeding from a custom claim path.
//
// What this demo shows:
//   1. Settings view — OIDC configuration overview (issuer, scopes, roles claim)
//   2. Login flow — redirect to OIDC provider's authorization page
//   3. Callback — ID token verification, role seeding, session creation
//   4. /auth/me — verifies the session carries the correct role
//   5. Deprecation path — Google env vars still work with a deprecation warning
//
// Prerequisites:
//   - Stack running: make compose-up (or port-forward to cluster)
//   - OIDC provider configured (or auth disabled for UI-only demo)
//   - STUDIO_API_TOKEN env var if testing with auth disabled
//
// Run:
//   cd demo && npx cypress run --no-runner-ui --spec 'cypress/e2e/generic-oidc-auth-demo.cy.ts'

// ---------------------------------------------------------------------------
// Inline helpers (self-contained)
// ---------------------------------------------------------------------------

const LONG = 1800;
const PAUSE = 900;
const SHORT = 400;
const TYPE_DELAY = 40;

function setupDemoHelpers(): void {
  cy.document().then((doc) => {
    if (doc.getElementById("__demo_cursor__")) return;

    const cursor = doc.createElement("div");
    cursor.id = "__demo_cursor__";
    Object.assign(cursor.style, {
      position: "fixed",
      width: "18px",
      height: "18px",
      borderRadius: "50%",
      background: "white",
      boxShadow: "0 0 0 2px rgba(0,0,0,0.4), 0 2px 8px rgba(0,0,0,0.5)",
      zIndex: "999999",
      pointerEvents: "none",
      transform: "translate(-50%, -50%)",
      transition: "left 0.4s cubic-bezier(0.4,0,0.2,1), top 0.4s cubic-bezier(0.4,0,0.2,1)",
      left: "960px",
      top: "540px",
    });
    doc.body.appendChild(cursor);

    const bar = doc.createElement("div");
    bar.id = "__demo_caption__";
    Object.assign(bar.style, {
      position: "fixed",
      top: "0",
      left: "0",
      right: "0",
      background: "rgba(15,15,20,0.88)",
      backdropFilter: "blur(6px)",
      color: "#e8e8f0",
      fontFamily: "ui-monospace, 'Cascadia Code', monospace",
      fontSize: "13px",
      padding: "7px 16px",
      zIndex: "1000000",
      pointerEvents: "none",
      letterSpacing: "0.01em",
      display: "none",
    });
    doc.body.appendChild(bar);
  });
}

function caption(text: string): void {
  cy.document().then((doc) => {
    const bar = doc.getElementById("__demo_caption__");
    if (!bar) return;
    bar.style.display = "block";
    bar.textContent = text;
  });
}

function clearCaption(): void {
  cy.document().then((doc) => {
    const bar = doc.getElementById("__demo_caption__");
    if (bar) bar.style.display = "none";
  });
}

function moveTo(selector: string): void {
  cy.get(selector).first().then(($el) => {
    const rect = $el[0].getBoundingClientRect();
    cy.document().then((doc) => {
      const cursor = doc.getElementById("__demo_cursor__");
      if (!cursor) return;
      cursor.style.left = `${rect.left + rect.width / 2}px`;
      cursor.style.top = `${rect.top + rect.height / 2}px`;
    });
  });
  cy.wait(SHORT);
}

function clickEffect(): void {
  cy.document().then((doc) => {
    const cursor = doc.getElementById("__demo_cursor__");
    if (!cursor) return;
    const x = parseFloat(cursor.style.left);
    const y = parseFloat(cursor.style.top);
    const ripple = doc.createElement("div");
    Object.assign(ripple.style, {
      position: "fixed",
      left: `${x}px`,
      top: `${y}px`,
      width: "0px",
      height: "0px",
      borderRadius: "50%",
      border: "3px solid #4f8ef7",
      transform: "translate(-50%, -50%)",
      zIndex: "999998",
      pointerEvents: "none",
      opacity: "1",
      transition: "width 0.5s ease-out, height 0.5s ease-out, opacity 0.5s ease-out",
    });
    doc.body.appendChild(ripple);
    requestAnimationFrame(() => {
      ripple.style.width = "60px";
      ripple.style.height = "60px";
      ripple.style.opacity = "0";
    });
    setTimeout(() => ripple.remove(), 600);
  });
}

function cursorClick(selector: string): void {
  moveTo(selector);
  cy.wait(PAUSE);
  clickEffect();
  cy.get(selector).first().click();
  cy.wait(SHORT);
}

// ---------------------------------------------------------------------------
// Demo
// ---------------------------------------------------------------------------

describe("Generic OIDC Auth: provider-agnostic login and role seeding", () => {
  before(() => {
    const token = Cypress.env("STUDIO_API_TOKEN") as string | undefined;
    if (token) cy.setCookie("studio_session", token);
    cy.visit("/");
    cy.wait(1000);
    setupDemoHelpers();
  });

  it("demonstrates OIDC configuration and auth behaviour", () => {

    // -----------------------------------------------------------------------
    // Step 1: Orient — show the Studio login screen
    // -----------------------------------------------------------------------
    caption("Generic OIDC Auth — Provider-Agnostic Login Demo");
    cy.wait(LONG);

    caption("Step 1: Studio now supports any OIDC provider — not just Google");
    cy.wait(PAUSE);

    // If auth is enabled, the login screen shows.
    // If auth is disabled (dev mode), we land on the app directly.
    cy.url().then((url) => {
      if (url.includes("/auth/login") || cy.$$(".login-screen").length > 0) {
        caption("Login screen — 'Login' redirects to the configured OIDC issuer");
        cy.wait(LONG);
        caption("OIDC_ISSUER_URL determines the provider: Keycloak, Okta, Azure AD, or Google");
        cy.wait(LONG);
      }
    });

    // -----------------------------------------------------------------------
    // Step 2: Settings — show the current user and role after login
    // -----------------------------------------------------------------------
    caption("Step 2: After OIDC callback — verify user role via /auth/me");
    cy.wait(PAUSE);

    cy.request({ url: "/auth/me", failOnStatusCode: false }).then((resp) => {
      if (resp.status === 200) {
        const role = resp.body.role ?? "unknown";
        const email = resp.body.email ?? "unknown";
        caption(`Authenticated as ${email} — role: ${role}`);
        cy.wait(LONG);
      } else {
        caption("/auth/me → 401 (auth disabled or not logged in — expected in dev mode)");
        cy.wait(LONG);
      }
    });

    // -----------------------------------------------------------------------
    // Step 3: Navigate to Settings to show user management
    // -----------------------------------------------------------------------
    caption("Step 3: Settings — user management powered by OIDC identity (sub + issuer keyed)");
    cy.wait(PAUSE);

    // Navigate to settings via sidebar
    cy.get(".sidebar-settings-btn").first().then(($btn) => {
      moveTo(".sidebar-settings-btn");
      cy.wait(PAUSE);
      clickEffect();
      $btn.trigger("click");
    });
    cy.wait(PAUSE);

    // -----------------------------------------------------------------------
    // Step 4: Show OIDC-specific configuration notes
    // -----------------------------------------------------------------------
    caption("Step 4: Helm values — auth.oidc.* section replaces auth.google.*");
    cy.wait(LONG);

    // Inject a visual callout overlay describing the config
    cy.document().then((doc) => {
      const card = doc.createElement("div");
      Object.assign(card.style, {
        position: "fixed",
        bottom: "60px",
        right: "40px",
        background: "rgba(20,24,36,0.95)",
        border: "1px solid #4f8ef7",
        borderRadius: "8px",
        padding: "16px 20px",
        color: "#e8e8f0",
        fontFamily: "ui-monospace, 'Cascadia Code', monospace",
        fontSize: "12px",
        lineHeight: "1.6",
        zIndex: "999997",
        maxWidth: "360px",
        pointerEvents: "none",
      });
      card.innerHTML = [
        "<b style='color:#4f8ef7'>auth.oidc.issuerUrl</b>: https://keycloak.example.com/realms/studio<br>",
        "<b style='color:#4f8ef7'>auth.oidc.rolesClaim</b>: realm_access.roles<br>",
        "<b style='color:#4f8ef7'>auth.oidc.bootstrapEmails</b>: [admin@example.com]<br>",
        "<span style='color:#888'>auth.google.*: ⚠ deprecated → maps to OIDC_* with warning</span>",
      ].join("");
      doc.body.appendChild(card);

      cy.wait(LONG * 2);
      card.remove();
    });

    caption("PKCE (S256) is always on — no additional configuration required");
    cy.wait(LONG);

    // -----------------------------------------------------------------------
    // Step 5: Role seeding explanation
    // -----------------------------------------------------------------------
    caption("Step 5: Role seeding — new users seed from JWT claims, then DB is authoritative");
    cy.wait(PAUSE);

    cy.document().then((doc) => {
      const card = doc.createElement("div");
      Object.assign(card.style, {
        position: "fixed",
        bottom: "60px",
        left: "40px",
        background: "rgba(20,24,36,0.95)",
        border: "1px solid #4f8ef7",
        borderRadius: "8px",
        padding: "16px 20px",
        color: "#e8e8f0",
        fontFamily: "ui-monospace, 'Cascadia Code', monospace",
        fontSize: "12px",
        lineHeight: "1.8",
        zIndex: "999997",
        maxWidth: "400px",
        pointerEvents: "none",
      });
      card.innerHTML = [
        "<b>New user login flow:</b><br>",
        "1. Verify ID token (RS256/384/512 via JWKS)<br>",
        "2. If bootstrap allowlist set → only listed emails → admin<br>",
        "3. If JWT claim contains 'admin' + email verified → admin<br>",
        "4. If no admins exist + email verified → first-admin<br>",
        "5. Otherwise → reviewer<br>",
        "<br><b>Returning user:</b> DB role is authoritative (no JWT override)",
      ].join("");
      doc.body.appendChild(card);

      cy.wait(LONG * 2);
      card.remove();
    });

    clearCaption();
    cy.wait(LONG);
  });
});
