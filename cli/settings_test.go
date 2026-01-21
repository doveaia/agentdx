package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasAgentdxHooks_NoHooks(t *testing.T) {
	settings := &ClaudeSettings{
		EnabledPlugins: map[string]bool{
			"gopls-lsp@claude-plugins-official": true,
		},
	}

	assert.False(t, hasAgentdxHooks(settings))
}

func TestHasAgentdxHooks_EmptyHooks(t *testing.T) {
	settings := &ClaudeSettings{
		EnabledPlugins: map[string]bool{
			"gopls-lsp@claude-plugins-official": true,
		},
		Hooks: &SettingsHooks{},
	}

	assert.False(t, hasAgentdxHooks(settings))
}

func TestHasAgentdxHooks_PartialHooks_ReturnsFalse(t *testing.T) {
	settings := &ClaudeSettings{
		Hooks: &SettingsHooks{
			PreToolUse: []ToolHook{
				{
					Matcher: "Grep",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: "echo '⚠️ AGENTDX FALLBACK: Grep tool requested.'",
						},
					},
				},
			},
		},
	}

	// Should return false because not all required hooks are present (missing Glob and Bash)
	assert.False(t, hasAgentdxHooks(settings))
}

func TestHasAgentdxHooks_AllHooks_ReturnsTrue(t *testing.T) {
	settings := &ClaudeSettings{
		Hooks: &SettingsHooks{
			UserPromptSubmit: []ToolHook{
				{
					Matcher: "",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: ".claude/hooks/agentdx/agentdx-session-start.sh",
						},
					},
				},
			},
			PreToolUse: []ToolHook{
				{
					Matcher: "Grep",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: "echo '⚠️ AGENTDX FALLBACK: Grep tool requested.'",
						},
					},
				},
				{
					Matcher: "Glob",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: "echo '⚠️ AGENTDX FALLBACK: Glob tool requested.'",
						},
					},
				},
			},
			PostToolUse: []ToolHook{
				{
					Matcher: "Bash",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: ".claude/hooks/agentdx/agentdx-fallback.sh",
						},
					},
				},
			},
			Stop: []ToolHook{
				{
					Matcher: "",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: ".claude/hooks/agentdx/agentdx-session-stop.sh",
						},
					},
				},
			},
		},
	}

	// All hooks present - should return true
	assert.True(t, hasAgentdxHooks(settings))
}

func TestHasAgentdxHooks_OnlyPostToolUseFallback_ReturnsFalse(t *testing.T) {
	settings := &ClaudeSettings{
		Hooks: &SettingsHooks{
			PostToolUse: []ToolHook{
				{
					Matcher: "Bash",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: ".claude/hooks/agentdx/agentdx-fallback.sh",
						},
					},
				},
			},
		},
	}

	// Should return false because PreToolUse hooks are missing
	assert.False(t, hasAgentdxHooks(settings))
}

func TestHasAgentdxHooks_WithOtherHooks(t *testing.T) {
	settings := &ClaudeSettings{
		Hooks: &SettingsHooks{
			PreToolUse: []ToolHook{
				{
					Matcher: "SomeOtherTool",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: "echo 'This is some other tool hook'",
						},
					},
				},
			},
		},
	}

	assert.False(t, hasAgentdxHooks(settings))
}

// TestBehavior1_NoHooksObject tests merging when no hooks object is present
func TestBehavior1_NoHooksObject(t *testing.T) {
	input := `{
  "enabledPlugins": {
    "gopls-lsp@claude-plugins-official": true
  }
}`

	settings, err := parseSettings([]byte(input))
	require.NoError(t, err)

	// Verify no agentdx hooks initially
	assert.False(t, hasAgentdxHooks(settings))

	// Merge agentdx hooks
	merged := mergeAgentdxHooks(settings)

	// Verify merged settings have agentdx hooks
	assert.True(t, hasAgentdxHooks(merged))
	assert.NotNil(t, merged.Hooks)
	assert.Len(t, merged.Hooks.UserPromptSubmit, 1) // Session start (runs per user message)
	assert.Len(t, merged.Hooks.PreToolUse, 2)       // Grep and Glob warnings only
	assert.Len(t, merged.Hooks.PostToolUse, 1)      // Bash (fallback)
	// NOTE: No Stop hooks - daemon keeps running for fresh index

	// Verify enabledPlugins is preserved
	assert.True(t, merged.EnabledPlugins["gopls-lsp@claude-plugins-official"])

	// Verify output is valid JSON
	output, err := serializeSettings(merged)
	require.NoError(t, err)
	assert.NoError(t, validateSettingsJSON(output))

	// Verify specific hooks
	assert.Equal(t, "Grep", merged.Hooks.PreToolUse[0].Matcher)
	assert.Equal(t, "Glob", merged.Hooks.PreToolUse[1].Matcher)
	assert.Equal(t, "Bash", merged.Hooks.PostToolUse[0].Matcher)
}

// TestBehavior2_EmptyHooksObject tests merging when hooks object is present but empty
func TestBehavior2_EmptyHooksObject(t *testing.T) {
	input := `{
  "enabledPlugins": {
    "gopls-lsp@claude-plugins-official": true
  },
  "hooks": {
  }
}`

	settings, err := parseSettings([]byte(input))
	require.NoError(t, err)

	// Verify no agentdx hooks initially
	assert.False(t, hasAgentdxHooks(settings))

	// Merge agentdx hooks
	merged := mergeAgentdxHooks(settings)

	// Verify merged settings have agentdx hooks
	assert.True(t, hasAgentdxHooks(merged))
	assert.Len(t, merged.Hooks.UserPromptSubmit, 1) // Session start (runs per user message)
	assert.Len(t, merged.Hooks.PreToolUse, 2)       // Grep and Glob warnings only
	assert.Len(t, merged.Hooks.PostToolUse, 1)      // Bash (fallback)
	// NOTE: No Stop hooks - daemon keeps running for fresh index

	// Verify output is valid JSON
	output, err := serializeSettings(merged)
	require.NoError(t, err)
	assert.NoError(t, validateSettingsJSON(output))
}

// TestBehavior3_ExistingHooks tests merging when hooks with PreToolUse/PostToolUse already exist
func TestBehavior3_ExistingHooks(t *testing.T) {
	input := `{
  "enabledPlugins": {
    "gopls-lsp@claude-plugins-official": true
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "SomeOtherTool",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'This is some other tool hook'"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "AnotherTool",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'This is another tool hook'"
          }
        ]
      }
    ]
  }
}`

	settings, err := parseSettings([]byte(input))
	require.NoError(t, err)

	// Verify no agentdx hooks initially
	assert.False(t, hasAgentdxHooks(settings))

	// Verify existing hooks
	assert.Len(t, settings.Hooks.PreToolUse, 1)
	assert.Len(t, settings.Hooks.PostToolUse, 1)

	// Merge agentdx hooks
	merged := mergeAgentdxHooks(settings)

	// Verify merged settings have agentdx hooks appended
	assert.True(t, hasAgentdxHooks(merged))
	assert.Len(t, merged.Hooks.UserPromptSubmit, 1) // Session start (runs per user message)
	assert.Len(t, merged.Hooks.PreToolUse, 3)       // Original + Grep + Glob
	assert.Len(t, merged.Hooks.PostToolUse, 2)      // Original + Bash (fallback)
	// NOTE: No Stop hooks - daemon keeps running for fresh index

	// Verify original hooks are preserved at the beginning
	assert.Equal(t, "SomeOtherTool", merged.Hooks.PreToolUse[0].Matcher)
	assert.Equal(t, "Grep", merged.Hooks.PreToolUse[1].Matcher)
	assert.Equal(t, "Glob", merged.Hooks.PreToolUse[2].Matcher)

	assert.Equal(t, "AnotherTool", merged.Hooks.PostToolUse[0].Matcher)
	assert.Equal(t, "Bash", merged.Hooks.PostToolUse[1].Matcher)

	// Verify output is valid JSON
	output, err := serializeSettings(merged)
	require.NoError(t, err)
	assert.NoError(t, validateSettingsJSON(output))
}

// TestAlreadyHasAgentdxHooks tests that we detect when ALL agentdx hooks are present
func TestAlreadyHasAgentdxHooks(t *testing.T) {
	settings := &ClaudeSettings{
		Hooks: &SettingsHooks{
			UserPromptSubmit: []ToolHook{
				{
					Matcher: "",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: ".claude/hooks/agentdx/agentdx-session-start.sh",
						},
					},
				},
			},
			PreToolUse: []ToolHook{
				{
					Matcher: "Grep",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: "echo '⚠️ AGENTDX FALLBACK: Grep tool requested.'",
						},
					},
				},
				{
					Matcher: "Glob",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: "echo '⚠️ AGENTDX FALLBACK: Glob tool requested.'",
						},
					},
				},
			},
			PostToolUse: []ToolHook{
				{
					Matcher: "Bash",
					Hooks: []HookAction{
						{
							Type:    "command",
							Command: ".claude/hooks/agentdx/agentdx-fallback.sh",
						},
					},
				},
			},
			// NOTE: No Stop hooks - daemon keeps running for fresh index
		},
	}

	assert.True(t, hasAgentdxHooks(settings))
}

func TestCreateDefaultSettings(t *testing.T) {
	settings := createDefaultSettings()

	assert.NotNil(t, settings.Hooks)
	assert.Len(t, settings.Hooks.UserPromptSubmit, 1) // Session start (runs per user message, not per tool)
	assert.Len(t, settings.Hooks.PreToolUse, 2)       // Grep and Glob warnings only
	assert.Len(t, settings.Hooks.PostToolUse, 1)      // Bash (fallback)
	// NOTE: No Stop hooks - daemon keeps running for fresh index

	// Verify it has agentdx hooks
	assert.True(t, hasAgentdxHooks(settings))

	// Verify it serializes to valid JSON
	output, err := serializeSettings(settings)
	require.NoError(t, err)
	assert.NoError(t, validateSettingsJSON(output))
}

func TestParseSettings_InvalidJSON(t *testing.T) {
	input := `{invalid json}`

	_, err := parseSettings([]byte(input))
	assert.Error(t, err)
}

func TestValidateSettingsJSON_Valid(t *testing.T) {
	input := `{"hooks": {}}`
	assert.NoError(t, validateSettingsJSON([]byte(input)))
}

func TestValidateSettingsJSON_Invalid(t *testing.T) {
	input := `{invalid}`
	assert.Error(t, validateSettingsJSON([]byte(input)))
}

func TestSerializeSettings_RoundTrip(t *testing.T) {
	original := &ClaudeSettings{
		EnabledPlugins: map[string]bool{
			"plugin1": true,
			"plugin2": false,
		},
		Hooks: &SettingsHooks{
			PreToolUse: []ToolHook{
				{
					Matcher: "TestTool",
					Hooks: []HookAction{
						{Type: "command", Command: "test"},
					},
				},
			},
		},
	}

	// Serialize
	data, err := serializeSettings(original)
	require.NoError(t, err)

	// Parse back
	parsed, err := parseSettings(data)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, original.EnabledPlugins, parsed.EnabledPlugins)
	assert.Len(t, parsed.Hooks.PreToolUse, 1)
	assert.Equal(t, "TestTool", parsed.Hooks.PreToolUse[0].Matcher)
}

func TestMergeAgentdxHooks_DoesNotModifyOriginal(t *testing.T) {
	original := &ClaudeSettings{
		Hooks: &SettingsHooks{
			PreToolUse: []ToolHook{
				{Matcher: "Original", Hooks: []HookAction{}},
			},
		},
	}

	merged := mergeAgentdxHooks(original)

	// Original should still have only 1 hook
	assert.Len(t, original.Hooks.PreToolUse, 1)
	assert.Equal(t, "Original", original.Hooks.PreToolUse[0].Matcher)

	// Merged should have 3 PreToolUse hooks (Original + Grep + Glob)
	assert.Len(t, merged.Hooks.PreToolUse, 3)
	// And 1 UserPromptSubmit hook for session start
	assert.Len(t, merged.Hooks.UserPromptSubmit, 1)
}

func TestOutputFormat(t *testing.T) {
	// Test that output JSON is properly formatted
	settings := createDefaultSettings()

	output, err := serializeSettings(settings)
	require.NoError(t, err)

	// Verify JSON can be re-parsed
	var parsed map[string]any
	err = json.Unmarshal(output, &parsed)
	require.NoError(t, err)

	// Verify hooks structure exists
	hooks, ok := parsed["hooks"].(map[string]any)
	require.True(t, ok)

	userPromptSubmit, ok := hooks["UserPromptSubmit"].([]any)
	require.True(t, ok)
	assert.Len(t, userPromptSubmit, 1) // Session start

	preToolUse, ok := hooks["PreToolUse"].([]any)
	require.True(t, ok)
	assert.Len(t, preToolUse, 2) // Grep and Glob warnings only

	postToolUse, ok := hooks["PostToolUse"].([]any)
	require.True(t, ok)
	assert.Len(t, postToolUse, 1)

	// NOTE: No Stop hooks - daemon keeps running for fresh index
	// Verify Stop is not present or empty
	stopHooks, ok := hooks["Stop"]
	if ok {
		stopArr, isArr := stopHooks.([]any)
		if isArr {
			assert.Len(t, stopArr, 0, "Stop hooks should be empty")
		}
	}
}
