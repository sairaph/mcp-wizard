# Agentic Tool Design Conventions

Generalised patterns for designing MCP tool surfaces, extracted from
interactive-terminal-mcp, sana-mcp, and favro-mcp.

---

## Tool Architecture

### Dedicated Tools (preferred)

Each operation is a separate MCP tool with its own name, input schema, and
handler. Models can discover tools by name without reading an action registry.

```
list_boards, get_card_details, create_card, delete_card, ...
it_list, it_new, it_read, it_send, it_kill, it_tail, it_head
```

### Combined Tool with Action Dispatch (alternative)

One MCP tool that dispatches on an action/operation name. Useful when all
operations share a single auth/session/context.

```
meeting_transcripts({tool: "list"})
meeting_transcripts({tool: "read", args: {meeting_id: "..."}})
```

### Naming

| Element | Convention | Example |
|---|---|---|
| Tool name | snake_case, verb-prefixed | `list_boards`, `create_card` |
| Read tools | `list_<plural>` / `get_<singular>` | `list_boards`, `get_card` |
| Write tools | `create_<entity>` / `update_<entity>` | `create_card`, `update_card` |
| Delete tools | `delete_<entity>` / `remove_<entity>` | `delete_card`, `remove_tag` |
| Argument names | snake_case | `meeting_id`, `board_id` |
| Optional args | Pointer types with omitempty | `*string`, `*bool` |

---

## Output Format

### YAML Frontmatter + Markdown Body (preferred)

```yaml
---
field: value
list:
  - item: value
---
Markdown body with guidance and next steps.
```

### Rules

- YAML from typed Go structs, not `map[string]any` (non-deterministic key order)
- `omitempty` on all optional fields
- Names listed first in rows, then IDs
- Body includes state-aware guidance with copy-pasteable next tool calls
- Hard ceiling: 1 MiB per response

---

## Error Contract

```yaml
error:
  code: not_found
  message: card not found: zzz-nonexistent
  hint: Call the relevant `list_*` tool to see valid IDs and names.
```

### Error Codes

| Code | Meaning |
|---|---|
| `not_found` | Resource does not exist |
| `invalid_input` | Argument validation failed |
| `authentication` | Credentials rejected |
| `forbidden` | Access denied |
| `rate_limited` | API throttled |
| `ambiguous` | Name matches multiple entities |
| `conflict` | Resource state conflict |
| `unavailable` | Service/dependency down |
| `internal_error` | Unexpected failure |

### Rules

- Every error carries a `hint` with a concrete, copy-pasteable next tool call
- Errors are returned as structured results with `IsError: true`
- Classify domain errors into typed codes
- Validation errors normalised from SDK framing into clean prose
- Unknown arguments rejected at schema level
- Forward-facing only - no changelog phrasing

---

## Description Style

- Multi-sentence paragraphs (120+ characters)
- Explain when to use the tool, what it returns, and how it relates to others
- Name related tools explicitly
- Explain edge cases (empty results, rate limiting, pagination)
- Forward-facing only
- End with consistent suffix where domain-appropriate (e.g. "on Favro.")
