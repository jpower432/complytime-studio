/**
 * Cupcake OpenCode Plugin
 * 
 * Install: Copy this file to .opencode/plugin/cupcake.js
 * 
 * This plugin integrates Cupcake policy enforcement with OpenCode.
 * It intercepts tool executions and evaluates them against your policies.
 */

// src/types.ts
var DEFAULT_CONFIG = {
  enabled: true,
  cupcakePath: "cupcake",
  harness: "opencode",
  logLevel: "warn",
  // Default to warn - info/debug are noisy in TUI
  timeoutMs: 5e3,
  failMode: "closed",
  cacheDecisions: false,
  showToasts: true,
  toastDurationMs: 5e3
};
function getToastVariant(decision) {
  switch (decision) {
    case "allow":
      return "success";
    case "ask":
      return "warning";
    case "deny":
    case "block":
      return "error";
    default:
      return "info";
  }
}

// src/event-builder.ts
function normalizeTool(tool) {
  return tool;
}
function buildPreToolUseEvent(sessionId, cwd, tool, args, agent, messageId, callId) {
  const event = {
    hook_event_name: "PreToolUse",
    session_id: sessionId,
    cwd,
    tool: normalizeTool(tool),
    args
  };
  if (agent) {
    event.agent = agent;
  }
  if (messageId) {
    event.message_id = messageId;
  }
  if (callId) {
    event.call_id = callId;
  }
  return event;
}
function buildPermissionEvent(sessionId, cwd, permissionId, permissionType, title, metadata, pattern, messageId, callId) {
  const event = {
    hook_event_name: "PermissionRequest",
    session_id: sessionId,
    cwd,
    permission_id: permissionId,
    permission_type: permissionType,
    title,
    metadata
  };
  if (pattern) {
    event.pattern = pattern;
  }
  if (messageId) {
    event.message_id = messageId;
  }
  if (callId) {
    event.call_id = callId;
  }
  return event;
}

// src/executor.ts
async function executeCupcake(config, event) {
  const startTime = Date.now();
  const eventJson = JSON.stringify(event);
  if (config.logLevel === "debug") {
    console.error(`[cupcake] DEBUG: Executing cupcake`);
    console.error(`[cupcake] DEBUG: Event:`, eventJson);
  }
  const proc = Bun.spawn([config.cupcakePath, "eval", "--harness", config.harness], {
    stdin: "pipe",
    stdout: "pipe",
    stderr: "ignore"
  });
  proc.stdin.write(eventJson);
  proc.stdin.end();
  const timeoutPromise = new Promise((_, reject) => {
    setTimeout(() => {
      proc.kill();
      reject(
        new Error(
          `Policy evaluation timed out after ${config.timeoutMs}ms. Consider optimizing policies or increasing timeout.`
        )
      );
    }, config.timeoutMs);
  });
  try {
    const [stdout, exitCode] = await Promise.race([
      Promise.all([new Response(proc.stdout).text(), proc.exited]),
      timeoutPromise
    ]);
    const elapsed = Date.now() - startTime;
    if (config.logLevel === "debug") {
      console.error(`[cupcake] DEBUG: Cupcake response (${elapsed}ms):`, stdout);
    }
    if (exitCode !== 0) {
      const error = new Error(`Cupcake exited with code ${exitCode}`);
      if (config.failMode === "open") {
        console.error(`[cupcake] ERROR: ${error.message}`);
        console.error(`[cupcake] WARN: Allowing operation in fail-open mode.`);
        return { decision: "allow" };
      }
      throw error;
    }
    const response = JSON.parse(stdout);
    if (config.logLevel === "debug") {
      console.error(`[cupcake] DEBUG: Decision: ${response.decision} (${elapsed}ms)`);
    }
    return response;
  } catch (error) {
    if (config.failMode === "open") {
      console.error(`[cupcake] ERROR: ${error.message}`);
      console.error(`[cupcake] WARN: Allowing operation in fail-open mode.`);
      return { decision: "allow" };
    }
    throw error;
  }
}

// src/enforcer.ts
function formatDecision(response) {
  const { decision, reason, rule_id, severity } = response;
  let title;
  let message;
  let blocked = false;
  switch (decision) {
    case "allow":
      title = "Allowed";
      message = reason || "Operation allowed by policy";
      break;
    case "deny":
    case "block":
      title = "Policy Violation";
      message = reason || `Operation blocked by policy`;
      blocked = true;
      break;
    case "ask":
      title = "Approval Required";
      message = reason || "This operation requires approval";
      blocked = true;
      break;
    default:
      title = "Unknown Decision";
      message = `Policy returned unknown decision: ${decision}`;
      blocked = true;
  }
  if (rule_id || severity) {
    const details = [];
    if (rule_id) details.push(`Rule: ${rule_id}`);
    if (severity) details.push(`Severity: ${severity}`);
    message += `
(${details.join(", ")})`;
  }
  return {
    blocked,
    title,
    message,
    variant: getToastVariant(decision),
    decision,
    ruleId: rule_id,
    severity
  };
}
function formatErrorMessage(formatted) {
  let message = "";
  if (formatted.decision === "deny" || formatted.decision === "block") {
    message += "\u274C Policy Violation\n\n";
  } else if (formatted.decision === "ask") {
    message += "\u26A0\uFE0F  Approval Required\n\n";
  }
  message += formatted.message;
  if (formatted.decision === "ask") {
    message += "\n\nNote: This operation requires manual approval. ";
    message += "To proceed, review the policy and temporarily disable it if appropriate, ";
    message += "then re-run the command.";
  }
  return message;
}

// src/index.ts
import { existsSync, readFileSync } from "fs";
import { join } from "path";
function loadConfig(directory) {
  const configPath = join(directory, ".cupcake", "opencode.json");
  if (existsSync(configPath)) {
    try {
      const configData = readFileSync(configPath, "utf-8");
      const userConfig = JSON.parse(configData);
      return { ...DEFAULT_CONFIG, ...userConfig };
    } catch (error) {
      console.error(`[cupcake] WARN: Failed to load config from ${configPath}: ${error.message}`);
      console.error(`[cupcake] WARN: Using default configuration`);
    }
  }
  return DEFAULT_CONFIG;
}
async function showToast(client, config, title, message, variant) {
  if (!config.showToasts || !client) {
    return;
  }
  try {
    await client.tui.showToast({
      body: {
        title,
        message,
        variant,
        duration: config.toastDurationMs
      }
    });
  } catch (error) {
    if (config.logLevel === "debug") {
      console.error(`[cupcake] DEBUG: Failed to show toast: ${error.message}`);
    }
  }
}
function log(config, level, message, ...args) {
  const levels = ["debug", "info", "warn", "error"];
  const configLevel = levels.indexOf(config.logLevel);
  const messageLevel = levels.indexOf(level);
  if (messageLevel >= configLevel) {
    const prefix = `[cupcake] ${level.toUpperCase()}:`;
    if (args.length > 0) {
      console.error(prefix, message, ...args);
    } else {
      console.error(prefix, message);
    }
  }
}
var CupcakePlugin = async ({ directory, client }) => {
  const config = loadConfig(directory);
  if (!config.enabled) {
    log(config, "debug", "Plugin is disabled in configuration");
    return {};
  }
  log(config, "debug", "Cupcake plugin initialized");
  return {
    /**
     * Hook: tool.execute.before
     *
     * Fired before any tool execution. This is where we enforce policies.
     * Throwing an error blocks the tool execution.
     */
    "tool.execute.before": async (input, output) => {
      try {
        log(config, "debug", `tool.execute.before fired for ${input.tool}`);
        log(config, "debug", "Args:", output.args);
        const event = buildPreToolUseEvent(
          input.sessionID || "unknown",
          directory,
          input.tool,
          output.args,
          void 0,
          // agent - not provided in current hook
          void 0,
          // messageId - not provided in current hook
          input.callID
        );
        const response = await executeCupcake(config, event);
        const formatted = formatDecision(response);
        if (formatted.decision !== "allow") {
          await showToast(client, config, formatted.title, formatted.message, formatted.variant);
        }
        if (formatted.blocked) {
          throw new Error(formatErrorMessage(formatted));
        }
        log(config, "debug", "Allowing tool execution");
      } catch (error) {
        throw error;
      }
    },
    /**
     * Hook: permission.ask
     *
     * Fired when OpenCode needs to request permission for an operation.
     * This integrates with OpenCode's native permission UI.
     *
     * - Set output.status = "allow" to auto-approve
     * - Set output.status = "deny" to auto-deny
     * - Leave as "ask" to show native permission dialog
     */
    "permission.ask": async (input, output) => {
      try {
        log(config, "debug", `permission.ask fired for ${input.type}`);
        log(config, "debug", "Permission:", input);
        const event = buildPermissionEvent(
          input.sessionID,
          directory,
          input.id,
          input.type,
          input.title,
          input.metadata,
          input.pattern,
          input.messageID,
          input.callID
        );
        const response = await executeCupcake(config, event);
        switch (response.decision) {
          case "allow":
            output.status = "allow";
            log(config, "debug", `Auto-allowing permission: ${input.type}`);
            break;
          case "deny":
          case "block":
            output.status = "deny";
            log(config, "debug", `Auto-denying permission: ${input.type}`);
            await showToast(
              client,
              config,
              "Permission Denied",
              response.reason || `Permission ${input.type} blocked by policy`,
              "error"
            );
            break;
          case "ask":
          default:
            output.status = "ask";
            log(config, "debug", `Deferring permission to user: ${input.type}`);
            if (response.reason) {
              await showToast(client, config, "Approval Recommended", response.reason, "warning");
            }
            break;
        }
      } catch (error) {
        log(config, "error", `Permission evaluation failed: ${error.message}`);
        output.status = "ask";
      }
    },
    /**
     * Hook: tool.execute.after
     *
     * Fired after tool execution. Used for audit logging.
     * Cannot prevent execution (already happened).
     */
    "tool.execute.after": async (input, output) => {
      log(config, "debug", `tool.execute.after fired for ${input.tool}`);
      log(config, "debug", "Output:", output.output?.substring(0, 200));
    },
    /**
     * Hook: event
     *
     * Fired for all OpenCode events. Used for comprehensive audit logging.
     */
    event: async ({ event }) => {
      if (config.logLevel !== "debug") {
        return;
      }
      const auditEvents = [
        "tool.executed",
        "permission.replied",
        "file.edited",
        "session.created",
        "session.aborted"
      ];
      if (auditEvents.includes(event.type)) {
        log(config, "debug", `Audit event: ${event.type}`, event.properties);
      }
    }
  };
};
export { CupcakePlugin };
