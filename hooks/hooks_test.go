package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestHooksDir(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	hooksDir := filepath.Join(cwd, AgentdxHooksDir)
	startDir := filepath.Join(hooksDir, "start")
	stopDir := filepath.Join(hooksDir, "stop")

	// Create test directories
	if err := os.MkdirAll(startDir, 0755); err != nil {
		t.Fatalf("Failed to create test hooks directory: %v", err)
	}
	if err := os.MkdirAll(stopDir, 0755); err != nil {
		t.Fatalf("Failed to create test hooks directory: %v", err)
	}

	// Create test hook scripts
	testScripts := map[string]string{
		"start/claude-code.sh": "#!/bin/sh\necho 'start claude-code'\nexit 0\n",
		"stop/claude-code.sh":  "#!/bin/sh\necho 'stop claude-code'\nexit 0\n",
		"start/codex.sh":       "#!/bin/sh\necho 'start codex'\nexit 0\n",
		"stop/codex.sh":        "#!/bin/sh\necho 'stop codex'\nexit 0\n",
		"start/opencode.sh":    "#!/bin/sh\necho 'start opencode'\nexit 0\n",
		"stop/opencode.sh":     "#!/bin/sh\necho 'stop opencode'\nexit 0\n",
	}

	for relPath, content := range testScripts {
		fullPath := filepath.Join(hooksDir, relPath)
		if err := os.WriteFile(fullPath, []byte(content), 0755); err != nil {
			t.Fatalf("Failed to create test script %s: %v", relPath, err)
		}
	}

	return hooksDir
}

func cleanupTestHooksDir(t *testing.T, hooksDir string) {
	t.Helper()
	os.RemoveAll(hooksDir)
}

func TestGetHookScript(t *testing.T) {
	hooksDir := setupTestHooksDir(t)
	defer cleanupTestHooksDir(t, hooksDir)

	tests := []struct {
		name        string
		agentName   string
		scriptType  string
		wantContent string
		wantErr     bool
	}{
		{
			name:        "claude-code start script",
			agentName:   "claude-code",
			scriptType:  "start",
			wantContent: "start claude-code",
			wantErr:     false,
		},
		{
			name:        "claude-code stop script",
			agentName:   "claude-code",
			scriptType:  "stop",
			wantContent: "stop claude-code",
			wantErr:     false,
		},
		{
			name:        "codex start script",
			agentName:   "codex",
			scriptType:  "start",
			wantContent: "start codex",
			wantErr:     false,
		},
		{
			name:        "non-existent script",
			agentName:   "unknown",
			scriptType:  "start",
			wantContent: "",
			wantErr:     true,
		},
		{
			name:        "invalid script type",
			agentName:   "claude-code",
			scriptType:  "invalid",
			wantContent: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := GetHookScript(tt.agentName, tt.scriptType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetHookScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(string(content), tt.wantContent) {
				t.Errorf("GetHookScript() content should contain %q", tt.wantContent)
			}
		})
	}
}

func TestListHookScripts(t *testing.T) {
	hooksDir := setupTestHooksDir(t)
	defer cleanupTestHooksDir(t, hooksDir)

	scripts, err := ListHookScripts()
	if err != nil {
		t.Fatalf("ListHookScripts() failed: %v", err)
	}

	// Check start scripts
	if len(scripts["start"]) != 3 {
		t.Errorf("ListHookScripts() returned %d start scripts, want 3", len(scripts["start"]))
	}

	// Check stop scripts
	if len(scripts["stop"]) != 3 {
		t.Errorf("ListHookScripts() returned %d stop scripts, want 3", len(scripts["stop"]))
	}

	// Verify expected agents
	expectedAgents := []string{"claude-code", "codex", "opencode"}
	for _, agent := range expectedAgents {
		foundStart := false
		foundStop := false
		for _, name := range scripts["start"] {
			if name == agent {
				foundStart = true
				break
			}
		}
		for _, name := range scripts["stop"] {
			if name == agent {
				foundStop = true
				break
			}
		}
		if !foundStart {
			t.Errorf("ListHookScripts() should have start script for %q", agent)
		}
		if !foundStop {
			t.Errorf("ListHookScripts() should have stop script for %q", agent)
		}
	}
}

func TestSupportedAgents(t *testing.T) {
	agents := SupportedAgents()

	if len(agents) != 3 {
		t.Errorf("SupportedAgents() returned %d agents, want 3", len(agents))
	}

	// Check claude-code
	claudeCodeFound := false
	for _, agent := range agents {
		if agent.Name == "claude-code" {
			claudeCodeFound = true
			if agent.StartScript != "claude-code.sh" {
				t.Errorf("claude-code StartScript = %s, want claude-code.sh", agent.StartScript)
			}
			if agent.StopScript != "claude-code.sh" {
				t.Errorf("claude-code StopScript = %s, want claude-code.sh", agent.StopScript)
			}
		}
	}
	if !claudeCodeFound {
		t.Error("SupportedAgents() should contain claude-code")
	}
}

func TestGetAgentConfig(t *testing.T) {
	tests := []struct {
		name    string
		agent   string
		wantErr bool
	}{
		{"claude-code", "claude-code", false},
		{"codex", "codex", false},
		{"opencode", "opencode", false},
		{"unknown", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := GetAgentConfig(tt.agent)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAgentConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if config.Name != tt.agent {
					t.Errorf("GetAgentConfig() Name = %s, want %s", config.Name, tt.agent)
				}
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"tilde at start", "~/test", filepath.Join(home, "test")},
		{"tilde only", "~", home},
		{"regular path", "/tmp/test", "/tmp/test"},
		{"relative path", "test", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandPath(tt.input)
			if err != nil {
				t.Fatalf("ExpandPath() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("ExpandPath() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetHookPath(t *testing.T) {
	// Get current working directory for project-scoped paths
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	claudeCode, err := GetAgentConfig("claude-code")
	if err != nil {
		t.Fatalf("Failed to get claude-code config: %v", err)
	}

	tests := []struct {
		name     string
		hookType string
		wantPath string
	}{
		{
			name:     "start hook",
			hookType: "start",
			wantPath: filepath.Join(cwd, ".claude/hooks/agentdx/agentdx-session-start.sh"),
		},
		{
			name:     "stop hook",
			hookType: "stop",
			wantPath: filepath.Join(cwd, ".claude/hooks/agentdx/agentdx-session-stop.sh"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetHookPath(claudeCode, tt.hookType)
			if err != nil {
				t.Fatalf("GetHookPath() failed: %v", err)
			}
			if got != tt.wantPath {
				t.Errorf("GetHookPath() = %s, want %s", got, tt.wantPath)
			}
		})
	}
}

func TestGetHookPath_InvalidType(t *testing.T) {
	claudeCode, err := GetAgentConfig("claude-code")
	if err != nil {
		t.Fatalf("Failed to get claude-code config: %v", err)
	}

	_, err = GetHookPath(claudeCode, "invalid")
	if err == nil {
		t.Error("GetHookPath() should return error for invalid hook type")
	}
}

// TestProjectScopedPaths verifies that hooks use project-relative paths, not global paths
func TestProjectScopedPaths(t *testing.T) {
	agents := SupportedAgents()

	for _, agent := range agents {
		t.Run(agent.Name, func(t *testing.T) {
			// Verify paths don't start with ~ (global)
			if strings.HasPrefix(agent.StartHookDir, "~") {
				t.Errorf("StartHookDir should not start with ~, got: %s", agent.StartHookDir)
			}
			if strings.HasPrefix(agent.StopHookDir, "~") {
				t.Errorf("StopHookDir should not start with ~, got: %s", agent.StopHookDir)
			}

			// Verify paths start with . (project-relative)
			if !strings.HasPrefix(agent.StartHookDir, ".") {
				t.Errorf("StartHookDir should start with . for project-relative path, got: %s", agent.StartHookDir)
			}
			if !strings.HasPrefix(agent.StopHookDir, ".") {
				t.Errorf("StopHookDir should start with . for project-relative path, got: %s", agent.StopHookDir)
			}
		})
	}
}

func TestHookScriptContent(t *testing.T) {
	hooksDir := setupTestHooksDir(t)
	defer cleanupTestHooksDir(t, hooksDir)

	// Verify that scripts have the required shebang and exit patterns
	scripts, err := ListHookScripts()
	if err != nil {
		t.Fatalf("ListHookScripts() failed: %v", err)
	}

	for _, scriptName := range scripts["start"] {
		t.Run(scriptName+"-start", func(t *testing.T) {
			content, err := GetHookScript(scriptName, "start")
			if err != nil {
				t.Fatalf("GetHookScript() failed: %v", err)
			}

			contentStr := string(content)

			// Check for shebang
			if !strings.HasPrefix(contentStr, "#!/bin/sh") {
				t.Error("Script should start with #!/bin/sh shebang")
			}

			// Check for exit 0 somewhere in the script
			if !strings.Contains(contentStr, "exit 0") {
				t.Error("Script should contain 'exit 0' for safety")
			}
		})
	}

	for _, scriptName := range scripts["stop"] {
		t.Run(scriptName+"-stop", func(t *testing.T) {
			content, err := GetHookScript(scriptName, "stop")
			if err != nil {
				t.Fatalf("GetHookScript() failed: %v", err)
			}

			contentStr := string(content)

			// Check for shebang
			if !strings.HasPrefix(contentStr, "#!/bin/sh") {
				t.Error("Script should start with #!/bin/sh shebang")
			}

			// Check for exit 0 somewhere in the script
			if !strings.Contains(contentStr, "exit 0") {
				t.Error("Script should contain 'exit 0' for safety")
			}
		})
	}
}

func TestEnsureAgentdxHooksDir(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "agentdx-hooks-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test creating new directory
	if err := EnsureAgentdxHooksDir(tmpDir); err != nil {
		t.Errorf("EnsureAgentdxHooksDir() failed: %v", err)
	}

	// Verify directories were created
	startDir := filepath.Join(tmpDir, AgentdxHooksDir, "start")
	stopDir := filepath.Join(tmpDir, AgentdxHooksDir, "stop")

	if info, err := os.Stat(startDir); err != nil || !info.IsDir() {
		t.Error("Start directory was not created")
	}
	if info, err := os.Stat(stopDir); err != nil || !info.IsDir() {
		t.Error("Stop directory was not created")
	}

	// Test idempotency - calling again should not error
	if err := EnsureAgentdxHooksDir(tmpDir); err != nil {
		t.Errorf("EnsureAgentdxHooksDir() should be idempotent, got error: %v", err)
	}
}
