#!/bin/sh
# agentdx session hook - stops watch daemon when coding agent session ends
# Installed by: agentdx setup
# Location: ./.claude/hooks/Stop/agentdx-session.sh (project-scoped)

# Only run if there's a session PID file
if [ ! -f ".agentdx/session.pid" ]; then
    exit 0
fi

# Stop the session daemon
agentdx session stop --quiet 2>/dev/null || true

# Always exit 0 to not block the coding agent
exit 0
