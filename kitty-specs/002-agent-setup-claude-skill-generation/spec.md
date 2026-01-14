# Feature Specification: Agent-Setup Claude Code Skill Generation

**Feature Branch**: `002-agent-setup-claude-skill-generation`
**Created**: 2026-01-14
**Status**: Draft
**Input**: User description: "Enhance agent-setup CLI command to generate Claude Code skill file at .claude/skills/agentdx/SKILL.md with dynamic content based on search type (semantic vs fulltext)"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Automatic Skill Generation on Agent Setup (Priority: P1)

As a developer using agentdx with Claude Code, when I run `agentdx agent-setup`, the command automatically creates a Claude Code skill file at `.claude/skills/agentdx/SKILL.md` so that Claude Code can leverage agentdx for code exploration.

**Why this priority**: This is the core feature - without automatic skill generation, users must manually create and maintain the skill file, which is error-prone and tedious.

**Independent Test**: Can be fully tested by running `agentdx agent-setup` in a project with agentdx initialized and verifying the skill file is created with appropriate content.

**Acceptance Scenarios**:

1. **Given** agentdx is initialized in a project, **When** user runs `agentdx agent-setup`, **Then** `.claude/skills/agentdx/SKILL.md` is created with valid skill content
2. **Given** skill file already exists with correct content, **When** user runs `agentdx agent-setup`, **Then** the existing file is not modified (idempotent)
3. **Given** `.claude/skills/agentdx/` directory does not exist, **When** user runs `agentdx agent-setup`, **Then** the directory structure is created automatically

---

### User Story 2 - Dynamic Content Based on Search Type (Priority: P1)

As a developer, when the skill file is generated, the content should match my configured search backend - semantic search instructions for Ollama/OpenAI embedders, or full-text search instructions for PostgreSQL backend.

**Why this priority**: Different backends have different optimal usage patterns. Providing incorrect instructions would confuse Claude Code and result in suboptimal searches.

**Independent Test**: Can be tested by changing the embedder.provider in config and verifying the skill file content changes accordingly.

**Acceptance Scenarios**:

1. **Given** config has `embedder.provider: ollama` or `embedder.provider: openai`, **When** skill is generated, **Then** skill contains semantic search instructions (natural language queries, intent-based search)
2. **Given** config has `embedder.provider: postgres`, **When** skill is generated, **Then** skill contains full-text search instructions (keyword-based, parallel searches)
3. **Given** skill content, **Then** it should NOT mention database backends like PostgreSQL (abstracted away from users)

---

### User Story 3 - Skill File Content Quality (Priority: P2)

As Claude Code using the agentdx skill, the skill file must provide clear, actionable instructions that help me understand when and how to use agentdx effectively.

**Why this priority**: Poor skill documentation leads to Claude Code misusing or ignoring agentdx, defeating the purpose of the integration.

**Independent Test**: Can be tested by reviewing the generated skill file and verifying it contains all required sections.

**Acceptance Scenarios**:

1. **Given** generated skill file, **Then** it contains frontmatter with `name: agentdx` and description
2. **Given** generated skill file, **Then** it contains "When to Invoke This Skill" section with clear triggers
3. **Given** generated skill file, **Then** it contains usage examples with `agentdx search` and `agentdx trace` commands
4. **Given** generated skill file, **Then** it contains fallback instructions for when agentdx is unavailable

---

### Edge Cases

- What happens when agentdx is not initialized? → Command should fail with helpful error message
- What happens when skill file exists but has outdated content? → Skip modification (user may have customized it)
- What happens when `.claude/` directory exists but not `skills/agentdx/`? → Create only missing subdirectories
- How does system handle file permission errors? → Display warning and continue with other operations

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST create `.claude/skills/agentdx/SKILL.md` when `agentdx agent-setup` is run
- **FR-002**: System MUST create parent directories (`.claude/skills/agentdx/`) if they don't exist
- **FR-003**: System MUST detect search type from config (`embedder.provider`) and generate appropriate content
- **FR-004**: System MUST be idempotent - not modify existing skill files that contain the skill marker
- **FR-005**: System MUST NOT mention database backends (PostgreSQL) in skill content - use abstract terms like "full-text search"
- **FR-006**: Generated skill MUST include frontmatter with `name` and `description` fields
- **FR-007**: Generated skill MUST include "When to Invoke This Skill" trigger conditions
- **FR-008**: Generated skill MUST include usage examples for `agentdx search` and `agentdx trace`
- **FR-009**: Generated skill MUST include fallback instructions for error scenarios
- **FR-010**: System MUST display status message when skill file is created or skipped

### Key Entities

- **SkillTemplate**: Template content for semantic or fulltext skill generation
- **SkillMarker**: Unique string to detect if skill is already configured (for idempotence)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Running `agentdx agent-setup` in any initialized project creates valid skill file
- **SC-002**: Generated skill file passes Claude Code skill validation (valid frontmatter, parseable markdown)
- **SC-003**: Command completes in under 1 second for skill generation
- **SC-004**: 100% of existing agent-setup functionality continues to work (backward compatible)
