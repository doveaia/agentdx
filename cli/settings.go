package cli

import (
	"encoding/json"
	"fmt"
)

// ClaudeSettings represents the .claude/settings.json structure
type ClaudeSettings struct {
	EnabledPlugins map[string]bool `json:"enabledPlugins,omitempty"`
	Hooks          *SettingsHooks  `json:"hooks,omitempty"`
}

// SettingsHooks contains hook configurations
type SettingsHooks struct {
	UserPromptSubmit []ToolHook `json:"UserPromptSubmit,omitempty"`
	PreToolUse       []ToolHook `json:"PreToolUse,omitempty"`
	PostToolUse      []ToolHook `json:"PostToolUse,omitempty"`
	Stop             []ToolHook `json:"Stop,omitempty"`
}

// ToolHook represents a single tool hook configuration
type ToolHook struct {
	Matcher string       `json:"matcher"`
	Hooks   []HookAction `json:"hooks"`
}

// HookAction represents an action within a hook
type HookAction struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// agentdxUserPromptSubmitHooks are the UserPromptSubmit hooks for session start
// This runs once per user message (not per tool call) - much better for session init
var agentdxUserPromptSubmitHooks = []ToolHook{
	{
		Matcher: "",
		Hooks: []HookAction{
			{
				Type:    "command",
				Command: ".claude/hooks/agentdx/agentdx-session-start.sh",
			},
		},
	},
}

// agentdxPreToolUseHooks are the PreToolUse hooks that agentdx needs
var agentdxPreToolUseHooks = []ToolHook{
	{
		Matcher: "Grep",
		Hooks: []HookAction{
			{
				Type:    "command",
				Command: "echo '⚠️ AGENTDX FALLBACK: Grep tool requested. Use agentdx search instead unless agentdx failed.'",
			},
		},
	},
	{
		Matcher: "Glob",
		Hooks: []HookAction{
			{
				Type:    "command",
				Command: "echo '⚠️ AGENTDX FALLBACK: Glob tool requested. Use agentdx files instead unless agentdx failed.'",
			},
		},
	},
}

// agentdxPostToolUseHooks are the PostToolUse hooks that agentdx needs
var agentdxPostToolUseHooks = []ToolHook{
	{
		Matcher: "Bash",
		Hooks: []HookAction{
			{
				Type:    "command",
				Command: ".claude/hooks/agentdx/agentdx-fallback.sh",
			},
		},
	},
}

// NOTE: Stop hooks removed - daemon should keep running to maintain fresh index
// Users can manually stop with: agentdx session stop

// hasAgentdxHooks checks if settings already contain ALL agentdx hooks
// Returns true only if all required agentdx hooks are present
func hasAgentdxHooks(settings *ClaudeSettings) bool {
	if settings.Hooks == nil {
		return false
	}

	// Check UserPromptSubmit hook for session start script
	hasSessionStartHook := false
	for _, hook := range settings.Hooks.UserPromptSubmit {
		for _, action := range hook.Hooks {
			if contains(action.Command, "agentdx-session-start.sh") {
				hasSessionStartHook = true
				break
			}
		}
	}

	// Check if all required PreToolUse hooks are present
	// We need hooks for both "Grep" and "Glob" matchers
	hasGrepHook := false
	hasGlobHook := false
	for _, hook := range settings.Hooks.PreToolUse {
		if hook.Matcher == "Grep" {
			for _, action := range hook.Hooks {
				if contains(action.Command, "AGENTDX FALLBACK") {
					hasGrepHook = true
					break
				}
			}
		}
		if hook.Matcher == "Glob" {
			for _, action := range hook.Hooks {
				if contains(action.Command, "AGENTDX FALLBACK") {
					hasGlobHook = true
					break
				}
			}
		}
	}

	// Check PostToolUse hook for "Bash" matcher
	hasBashHook := false
	for _, hook := range settings.Hooks.PostToolUse {
		if hook.Matcher == "Bash" {
			for _, action := range hook.Hooks {
				if contains(action.Command, "agentdx-fallback.sh") {
					hasBashHook = true
					break
				}
			}
		}
	}

	// All hooks must be present for configuration to be complete
	// NOTE: Stop hook no longer required - daemon keeps running for fresh index
	return hasSessionStartHook && hasGrepHook && hasGlobHook && hasBashHook
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mergeAgentdxHooks merges agentdx hooks into existing settings
// Returns the merged settings (does not modify the original)
// Avoids duplicates by checking if a hook with the same matcher already exists
func mergeAgentdxHooks(settings *ClaudeSettings) *ClaudeSettings {
	// Create a copy of the settings
	merged := &ClaudeSettings{
		EnabledPlugins: settings.EnabledPlugins,
	}

	// Initialize hooks if nil
	if settings.Hooks == nil {
		merged.Hooks = &SettingsHooks{
			UserPromptSubmit: make([]ToolHook, 0),
			PreToolUse:       make([]ToolHook, 0),
			PostToolUse:      make([]ToolHook, 0),
		}
	} else {
		merged.Hooks = &SettingsHooks{
			UserPromptSubmit: make([]ToolHook, 0, len(settings.Hooks.UserPromptSubmit)),
			PreToolUse:       make([]ToolHook, 0, len(settings.Hooks.PreToolUse)),
			PostToolUse:      make([]ToolHook, 0, len(settings.Hooks.PostToolUse)),
		}
		// Copy existing UserPromptSubmit hooks that are not agentdx hooks
		for _, hook := range settings.Hooks.UserPromptSubmit {
			if !isAgentdxSessionStartHook(hook) {
				merged.Hooks.UserPromptSubmit = append(merged.Hooks.UserPromptSubmit, hook)
			}
		}
		// Copy existing hooks, filtering out any agentdx hooks that will be replaced
		for _, hook := range settings.Hooks.PreToolUse {
			if !isAgentdxHookMatcher(hook.Matcher) {
				merged.Hooks.PreToolUse = append(merged.Hooks.PreToolUse, hook)
			}
		}
		for _, hook := range settings.Hooks.PostToolUse {
			if !isAgentdxHookMatcher(hook.Matcher) {
				merged.Hooks.PostToolUse = append(merged.Hooks.PostToolUse, hook)
			}
		}
		// NOTE: Stop hooks are removed from agentdx - daemon keeps running
		// Any existing user Stop hooks are preserved in the original settings
	}

	// Append agentdx UserPromptSubmit hooks (session start)
	merged.Hooks.UserPromptSubmit = append(merged.Hooks.UserPromptSubmit, agentdxUserPromptSubmitHooks...)

	// Append agentdx PreToolUse hooks
	merged.Hooks.PreToolUse = append(merged.Hooks.PreToolUse, agentdxPreToolUseHooks...)

	// Append agentdx PostToolUse hooks
	merged.Hooks.PostToolUse = append(merged.Hooks.PostToolUse, agentdxPostToolUseHooks...)

	return merged
}

// isAgentdxSessionStartHook checks if a hook is an agentdx session start hook
func isAgentdxSessionStartHook(hook ToolHook) bool {
	for _, action := range hook.Hooks {
		if contains(action.Command, "agentdx-session-start.sh") {
			return true
		}
	}
	return false
}

// isAgentdxHookMatcher checks if a matcher is used by agentdx hooks
func isAgentdxHookMatcher(matcher string) bool {
	agentdxMatchers := map[string]bool{
		"Grep": true,
		"Glob": true,
		"Bash": true,
	}
	return agentdxMatchers[matcher]
}

// parseSettings parses JSON bytes into ClaudeSettings
func parseSettings(data []byte) (*ClaudeSettings, error) {
	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings JSON: %w", err)
	}
	return &settings, nil
}

// serializeSettings converts ClaudeSettings to formatted JSON bytes
func serializeSettings(settings *ClaudeSettings) ([]byte, error) {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize settings JSON: %w", err)
	}
	return data, nil
}

// validateSettingsJSON validates that the JSON is well-formed
func validateSettingsJSON(data []byte) error {
	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// createDefaultSettings creates a new ClaudeSettings with agentdx hooks
func createDefaultSettings() *ClaudeSettings {
	return &ClaudeSettings{
		Hooks: &SettingsHooks{
			UserPromptSubmit: agentdxUserPromptSubmitHooks,
			PreToolUse:       agentdxPreToolUseHooks,
			PostToolUse:      agentdxPostToolUseHooks,
			// NOTE: No Stop hooks - daemon keeps running for fresh index
		},
	}
}
