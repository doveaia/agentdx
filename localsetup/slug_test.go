package localsetup

import "testing"

func TestToSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "myproject", "myproject"},
		{"with hyphen", "my-project", "my_project"},
		{"with spaces", "My Project", "my_project"},
		{"mixed case", "MyProject", "myproject"},
		{"with numbers", "project123", "project123"},
		{"numbers and hyphens", "123-numbers-first", "123_numbers_first"},
		{"special chars", "test@project!", "testproject"},
		{"unicode", "café-app", "caf_app"},
		{"multiple hyphens", "my--project", "my_project"},
		{"multiple spaces", "my   project", "my_project"},
		{"leading hyphen", "-project", "project"},
		{"trailing hyphen", "project-", "project"},
		{"underscores preserved", "my_project", "my_project"},
		{"mixed separators", "my-project_name", "my_project_name"},
		{"empty string", "", ""},
		{"only special chars", "!@#$%", ""},
		{"dots preserved", "my.project", "myproject"},
		{"at sign", "test@project", "testproject"},
		{"hash sign", "project#1", "project1"},
		{"dollar sign", "project$", "project"},
		{"percent sign", "project%", "project"},
		{"ampersand", "project&test", "projecttest"},
		{"asterisk", "project*test", "projecttest"},
		{"plus sign", "project+test", "projecttest"},
		{"equals sign", "project=test", "projecttest"},
		{"brackets", "project[test]", "projecttest"},
		{"braces", "project{test}", "projecttest"},
		{"pipe", "project|test", "projecttest"},
		{"backslash", "project\\test", "projecttest"},
		{"forward slash", "project/test", "projecttest"},
		{"question mark", "project?test", "projecttest"},
		{"less than", "project<test", "projecttest"},
		{"greater than", "project>test", "projecttest"},
		{"comma", "project,test", "projecttest"},
		{"semicolon", "project;test", "projecttest"},
		{"colon", "project:test", "projecttest"},
		{"single quote", "project'test", "projecttest"},
		{"double quote", "project\"test", "projecttest"},
		{"tab char", "project\ttest", "projecttest"},
		{"newline char", "project\ntest", "projecttest"},
		{"carriage return", "project\rtest", "projecttest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToSlug(tt.input)
			if got != tt.expected {
				t.Errorf("ToSlug(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
