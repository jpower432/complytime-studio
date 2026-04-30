// SPDX-License-Identifier: Apache-2.0
//
// Demo: SOC 2 Gap Analysis — AMPEL Branch Protection
//
// Walks through the full demo flow from demo/prompts.md:
//   1. Orient — show the AMPEL policy and its controls
//   2. Inventory — discover repositories and scan evidence
//   3. Gap Analysis — join evidence with SOC 2 CC8.1 / CC6.1
//   4. Drill Down — inspect branch protection failures on complytime-studio
//   5. Artifact — generate and save the Gemara AuditLog
//
// Prerequisites:
//   - Stack running: make compose-up (or port-forward to cluster)
//   - Demo data seeded: make seed
//   - STUDIO_API_TOKEN env var if auth is enabled
//
// Run:
//   cd demo && npm run demo

// ---------------------------------------------------------------------------
// Inline helpers (self-contained — no shared import needed)
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

function typeMessage(text: string): void {
  cy.get(".chat-overlay-input textarea, .chat-overlay-input input[type=text]")
    .first()
    .clear()
    .type(text, { delay: TYPE_DELAY });
  cy.wait(PAUSE);
}

function sendMessage(): void {
  clickEffect();
  cy.get(".chat-overlay-input .btn-primary").first().click();
}

/** Wait for the agent to finish streaming (thinking dot disappears). */
function waitForAgentResponse(timeout = 90000): void {
  cy.get(".chat-thinking", { timeout }).should("not.exist");
  cy.wait(LONG);
}

// ---------------------------------------------------------------------------
// Demo
// ---------------------------------------------------------------------------

describe("SOC 2 Gap Analysis — AMPEL Branch Protection", () => {
  before(() => {
    // Bypass auth for local demo if token is set
    const token = Cypress.env("STUDIO_API_TOKEN") as string | undefined;
    if (token) {
      cy.setCookie("studio_session", token);
    }
    cy.visit("/");
    cy.wait(1000);
    setupDemoHelpers();
  });

  it("walks through the full SOC 2 gap analysis demo", () => {

    // -----------------------------------------------------------------------
    // Step 1: Orient — open Posture view, open assistant
    // -----------------------------------------------------------------------
    caption("ComplyTime Studio — SOC 2 Gap Analysis Demo");
    cy.wait(LONG);

    caption("Step 1: Orient — open the AMPEL branch protection policy");
    cy.wait(PAUSE);

    // Navigate to Posture via sidebar
    cursorClick(".sidebar-item");
    cy.wait(PAUSE);

    // Open chat assistant
    caption("Opening the Studio Assistant...");
    cursorClick(".chat-fab");
    cy.wait(PAUSE);

    typeMessage("Show me the AMPEL branch protection policy and its controls.");
    caption("Asking the assistant to load the AMPEL policy from ClickHouse...");
    sendMessage();

    waitForAgentResponse();
    caption("Assistant loaded BP-1 through BP-5 controls. Policy context established.");
    cy.wait(LONG);

    // -----------------------------------------------------------------------
    // Step 2: Inventory — discover repos and evidence
    // -----------------------------------------------------------------------
    caption("Step 2: Inventory — what evidence do we have?");
    cy.wait(PAUSE);

    typeMessage("What evidence do we have for the ampel-branch-protection policy? Show me all targets.");
    caption("Discovering repositories scanned and evidence records...");
    sendMessage();

    waitForAgentResponse();
    caption("3 repositories · 45 evidence records · 3 scan dates (Apr 7, 14, 16)");
    cy.wait(LONG);

    // -----------------------------------------------------------------------
    // Step 3: SOC 2 Gap Analysis
    // -----------------------------------------------------------------------
    caption("Step 3: Gap Analysis — joining evidence with SOC 2 Trust Services Criteria");
    cy.wait(PAUSE);

    typeMessage("Run a SOC 2 gap analysis for policy ampel-branch-protection, audit period April 1-18 2026.");
    caption("Mapping evidence to CC8.1 (Change Management) and CC6.1 (Logical Access)...");
    sendMessage();

    waitForAgentResponse();
    caption("CC8.1: Not fully covered · CC6.1: At risk · complyctl: Clean ✓");
    cy.wait(LONG);

    // -----------------------------------------------------------------------
    // Step 4: Drill down — failures on complytime-studio
    // -----------------------------------------------------------------------
    caption("Step 4: Drill down — what exactly is failing on complytime-studio?");
    cy.wait(PAUSE);

    typeMessage("Show me the branch protection failures on complytime-studio. What's the risk?");
    caption("Inspecting BP-4 (admin bypass) and BP-5 (code owner review) failures...");
    sendMessage();

    waitForAgentResponse();
    caption("BP-4: Admin bypass enabled · BP-5: Code owner review missing · Persistent across all 3 scan dates");
    cy.wait(LONG);

    // -----------------------------------------------------------------------
    // Step 5: Generate the Gemara AuditLog artifact
    // -----------------------------------------------------------------------
    caption("Step 5: Generate and validate the Gemara AuditLog artifact");
    cy.wait(PAUSE);

    typeMessage("Generate the audit log for this analysis.");
    caption("Authoring validated Gemara #AuditLog YAML — 3 targets, 5 criteria each...");
    sendMessage();

    waitForAgentResponse(120000);
    caption("AuditLog generated: Findings on BP-4/BP-5 (studio), BP-2/BP-5 (policies) · Strength: complyctl");
    cy.wait(LONG);

    // Save artifact if the button appears
    cy.get("body").then(($body) => {
      if ($body.find(".chat-artifact-card .btn-primary").length > 0) {
        caption("Saving AuditLog artifact to the Audit workspace...");
        cursorClick(".chat-artifact-card .btn-primary");
        cy.wait(LONG);
        caption("Artifact saved. Navigate to Audit to review.");
      }
    });

    clearCaption();
    cy.wait(LONG);
  });
});
