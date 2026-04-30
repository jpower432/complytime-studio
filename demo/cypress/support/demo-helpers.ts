// SPDX-License-Identifier: Apache-2.0
//
// Shared demo helpers: synthetic cursor, click ripple, and caption overlay.
// Embed these directly in each demo spec — do NOT import from a shared path,
// so each spec is self-contained and runnable in isolation.
//
// Copy this file's helpers into your spec file, or import it explicitly:
//   import "../../support/demo-helpers";
//
// Usage: call setupDemoHelpers() inside your describe() before any tests.

export const LONG = 1800;    // pause to let audience read
export const PAUSE = 900;    // brief beat between actions
export const SHORT = 400;    // quick transition
export const TYPE_DELAY = 40; // ms between keystrokes (human feel)

/**
 * Inject the synthetic cursor and caption overlay into the page.
 * Call once inside a before() or at the start of your it() block.
 */
export function setupDemoHelpers(): void {
  cy.document().then((doc) => {
    // Cursor dot
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

    // Caption bar — pinned to top so it never obscures toolbars
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

/** Display a caption in the overlay bar. */
export function caption(text: string): void {
  cy.document().then((doc) => {
    const bar = doc.getElementById("__demo_caption__");
    if (!bar) return;
    bar.style.display = "block";
    bar.textContent = text;
  });
}

/** Hide the caption bar. */
export function clearCaption(): void {
  cy.document().then((doc) => {
    const bar = doc.getElementById("__demo_caption__");
    if (bar) bar.style.display = "none";
  });
}

/** Glide the synthetic cursor to a DOM element. */
export function moveTo(selector: string): void {
  cy.get(selector).first().then(($el) => {
    const rect = $el[0].getBoundingClientRect();
    const x = rect.left + rect.width / 2;
    const y = rect.top + rect.height / 2;
    cy.document().then((doc) => {
      const cursor = doc.getElementById("__demo_cursor__");
      if (!cursor) return;
      cursor.style.left = `${x}px`;
      cursor.style.top = `${y}px`;
    });
  });
  cy.wait(SHORT);
}

/** Glide the synthetic cursor to an element containing specific text. */
export function moveToText(text: string, tag = "*"): void {
  cy.contains(tag, text).first().then(($el) => {
    const rect = $el[0].getBoundingClientRect();
    const x = rect.left + rect.width / 2;
    const y = rect.top + rect.height / 2;
    cy.document().then((doc) => {
      const cursor = doc.getElementById("__demo_cursor__");
      if (!cursor) return;
      cursor.style.left = `${x}px`;
      cursor.style.top = `${y}px`;
    });
  });
  cy.wait(SHORT);
}

/** Emit a blue click-ripple at the current cursor position. */
export function clickEffect(): void {
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

    // Trigger expansion on next frame
    requestAnimationFrame(() => {
      ripple.style.width = "60px";
      ripple.style.height = "60px";
      ripple.style.opacity = "0";
    });

    setTimeout(() => ripple.remove(), 600);
  });
}

/**
 * High-level: move cursor to text, show ripple, then click the element.
 */
export function cursorClick(selector: string): void {
  moveTo(selector);
  cy.wait(PAUSE);
  clickEffect();
  cy.get(selector).first().click();
  cy.wait(SHORT);
}

/**
 * High-level: move cursor to text content, show ripple, then click.
 */
export function cursorClickText(text: string, tag = "*"): void {
  moveToText(text, tag);
  cy.wait(PAUSE);
  clickEffect();
  cy.contains(tag, text).first().click();
  cy.wait(SHORT);
}
