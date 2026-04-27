# SPDX-License-Identifier: Apache-2.0

# METADATA
# scope: package
# custom:
#   routing:
#     required_events: ["PreToolUse"]
#     required_tools: ["Bash"]
package cupcake.policies.opencode.gh_remote_write_guard

import rego.v1

deny contains decision if {
    input.tool_name == "Bash"
    regex.match(`\bgit\s+push\b`, input.tool_input.command)
    decision := {
        "rule_id": "GH-WRITE-001",
        "reason": "Remote write blocked: git push requires manual execution",
        "severity": "HIGH",
    }
}

deny contains decision if {
    input.tool_name == "Bash"
    regex.match(`\bgh\s+pr\s+(create|merge|close|comment|edit|review|ready|reopen)\b`, input.tool_input.command)
    decision := {
        "rule_id": "GH-WRITE-002",
        "reason": "Remote write blocked: gh pr write operations require manual execution",
        "severity": "HIGH",
    }
}

deny contains decision if {
    input.tool_name == "Bash"
    regex.match(`\bgh\s+issue\s+(create|close|comment|edit|reopen|delete|transfer|pin|unpin)\b`, input.tool_input.command)
    decision := {
        "rule_id": "GH-WRITE-003",
        "reason": "Remote write blocked: gh issue write operations require manual execution",
        "severity": "HIGH",
    }
}

deny contains decision if {
    input.tool_name == "Bash"
    regex.match(`\bgh\s+release\s+(create|delete|edit)\b`, input.tool_input.command)
    decision := {
        "rule_id": "GH-WRITE-004",
        "reason": "Remote write blocked: gh release write operations require manual execution",
        "severity": "HIGH",
    }
}

deny contains decision if {
    input.tool_name == "Bash"
    regex.match(`\bgh\s+repo\s+(create|delete|fork|rename|archive)\b`, input.tool_input.command)
    decision := {
        "rule_id": "GH-WRITE-005",
        "reason": "Remote write blocked: gh repo write operations require manual execution",
        "severity": "HIGH",
    }
}

deny contains decision if {
    input.tool_name == "Bash"
    regex.match(`\bgh\s+api\b`, input.tool_input.command)
    regex.match(`(-X|--method)\s+(POST|PUT|PATCH|DELETE)`, input.tool_input.command)
    decision := {
        "rule_id": "GH-WRITE-006",
        "reason": "Remote write blocked: gh api write methods require manual execution",
        "severity": "HIGH",
    }
}

deny contains decision if {
    input.tool_name == "Bash"
    regex.match(`\bgh\s+workflow\s+run\b`, input.tool_input.command)
    decision := {
        "rule_id": "GH-WRITE-007",
        "reason": "Remote write blocked: gh workflow run requires manual execution",
        "severity": "HIGH",
    }
}
