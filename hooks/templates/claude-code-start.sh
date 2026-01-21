#!/bin/sh
# agentdx session hook - starts watch daemon when coding agent session begins
# Installed by: agentdx setup
# Location: ./.claude/hooks/PreToolUse/agentdx-session.sh (project-scoped)

# Only run if this is an agentdx-initialized project
if [ ! -f ".agentdx/config.yaml" ]; then
    exit 0
fi

# Start the session daemon (idempotent - does nothing if already running)
agentdx session start --quiet 2>/dev/null || true

# Always exit 0 to not block the coding agent
exit 0
