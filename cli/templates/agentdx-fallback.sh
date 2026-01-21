#!/bin/bash
# PostToolUse hook for Bash tool
# Detects when agentdx returns empty results and instructs Claude to spawn Explore agent

# Read the hook input from stdin
INPUT=$(cat)

# Extract the command that was run
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# Extract the tool output (stdout)
OUTPUT=$(echo "$INPUT" | jq -r '.tool_output.stdout // empty')

# Check if this was an agentdx search command
if echo "$COMMAND" | grep -qE '^agentdx (search|files)'; then
  # Check if output is empty array [] (with optional whitespace)
  if echo "$OUTPUT" | grep -qE '^\s*\[\s*\]\s*$'; then
    # Extract the search query from the command
    QUERY=$(echo "$COMMAND" | sed -E 's/agentdx (search|files) "?([^"]*)"?.*/\2/')

    # Return JSON instructing Claude to spawn Explore agent
    cat << EOF
{
  "decision": "block",
  "reason": "agentdx returned empty results for '$QUERY'. You MUST now spawn the Explore agent to verify: Task(subagent_type=Explore, prompt='$QUERY')"
}
EOF
    exit 0
  fi
fi

# Default: approve the tool result
echo '{"decision": "approve"}'
exit 0
