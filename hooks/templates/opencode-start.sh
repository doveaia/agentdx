#!/bin/sh
# agentdx session hook for OpenCode
# TODO: Implement when OpenCode hook system is documented

if [ ! -f ".agentdx/config.yaml" ]; then
    exit 0
fi

agentdx session start --quiet 2>/dev/null || true
exit 0
