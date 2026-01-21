#!/bin/sh
# agentdx session hook for Codex CLI
# TODO: Implement when Codex hook system is documented

# Placeholder - same pattern as Claude Code hooks
if [ ! -f ".agentdx/config.yaml" ]; then
    exit 0
fi

agentdx session start --quiet 2>/dev/null || true
exit 0
