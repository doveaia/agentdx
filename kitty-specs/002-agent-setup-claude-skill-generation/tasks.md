# Tasks: Agent-Setup Claude Code Skill Generation

**Feature**: `002-agent-setup-claude-skill-generation`
**Created**: 2026-01-14

## Work Packages

### WP01: Add Skill Template Constants
**Status**: done
**Files**: `cli/agent_setup.go`

**Description**: Add two new template constants for Claude Code skill files - one for semantic search (ollama/openai) and one for full-text search (postgres). Also add the skill marker constant.

**Acceptance Criteria**:
- [ ] `semanticSkillTemplate` constant defined with valid frontmatter and semantic search instructions
- [ ] `fullTextSkillTemplate` constant defined with valid frontmatter and fulltext search instructions
- [ ] `skillMarker` constant defined as `"name: agentdx"`
- [ ] Neither template mentions PostgreSQL or database backends
- [ ] Both templates include: frontmatter, "When to Invoke" section, usage examples, fallback instructions

---

### WP02: Add createSkill Function
**Status**: done
**Files**: `cli/agent_setup.go`

**Description**: Create a new `createSkill` function following the pattern of existing `createSubagent` function. This function creates `.claude/skills/agentdx/SKILL.md` with the appropriate template content.

**Acceptance Criteria**:
- [ ] Function signature: `func createSkill(cwd string, skillTemplate string) error`
- [ ] Creates `.claude/skills/agentdx/` directory if it doesn't exist
- [ ] Checks for existing skill file with marker before writing (idempotent)
- [ ] Writes skill template to `.claude/skills/agentdx/SKILL.md`
- [ ] Prints status message: "Created skill: ..." or "Skill already exists: ..."
- [ ] Returns nil on success, error with context on failure

---

### WP03: Update getTemplates Function
**Status**: done
**Files**: `cli/agent_setup.go`

**Description**: Modify `getTemplates` function to return the appropriate skill template as an additional return value based on search type.

**Acceptance Criteria**:
- [ ] Function signature updated to return 5 values: `(instructions, subagent, marker, subagentMarker, skillTemplate string)`
- [ ] Returns `semanticSkillTemplate` when searchType is "semantic"
- [ ] Returns `fullTextSkillTemplate` when searchType is "fulltext"

---

### WP04: Update runAgentSetup to Create Skill
**Status**: done
**Files**: `cli/agent_setup.go`

**Description**: Modify `runAgentSetup` function to always call `createSkill()` after processing agent configuration files.

**Acceptance Criteria**:
- [ ] Calls `createSkill(cwd, skillTemplate)` unconditionally (always creates skill)
- [ ] Skill creation happens after agent config file processing
- [ ] Handles errors gracefully with warning message (doesn't fail whole command)
- [ ] Works correctly with or without `--with-subagent` flag

---

### WP05: Manual Testing and Verification
**Status**: done
**Files**: None (testing only)

**Description**: Verify the implementation works correctly across different configurations.

**Acceptance Criteria**:
- [ ] Test with ollama provider → semantic skill generated
- [ ] Test with openai provider → semantic skill generated
- [ ] Test with postgres provider → fulltext skill generated
- [ ] Test idempotence: second run doesn't overwrite
- [ ] Test in project without `.claude/` directory → directory created
- [ ] Verify skill file has valid YAML frontmatter
- [ ] Verify all existing `agent-setup` functionality still works

---

## Dependency Order

```
WP01 (templates) → WP03 (getTemplates update) → WP02 (createSkill) → WP04 (integration) → WP05 (testing)
```

## Estimated Complexity

- **WP01**: Medium (writing comprehensive template content)
- **WP02**: Low (follows existing pattern)
- **WP03**: Low (simple modification)
- **WP04**: Low (single function call)
- **WP05**: Low (manual verification)
