

<!-- copilot-mem:managed -->
# 🧠 Copilot Memory Protocol (managed by copilot-mem extension)

You have access to a **persistent long-term memory system** via MCP tools.
This memory survives across ALL sessions, days, weeks, and months.
Every workspace has its own memory store.

## RULE 1 — Session Start (MANDATORY, NO EXCEPTIONS)

At the very beginning of every conversation, run BOTH of these:

```
memory_recent({ limit: 10 })
```
This shows what was worked on recently.

```
memory_search({ query: "<topic from user's first message>" })
```
This retrieves relevant context before you respond.

## RULE 2 — Save personal information IMMEDIATELY (no prompting needed)

If the user mentions ANY personal information in their messages, 
save it RIGHT AWAY without being asked:

| User says | You do |
|-----------|--------|
| "my daughter is named Leo" | `memory_save({ kind: "observation", title: "Personal: daughter name", content: "User's daughter is named Leo", facts: ["daughter name: Leo"] })` |
| "I prefer TypeScript over JavaScript" | `memory_save({ kind: "observation", title: "Personal: language preference", content: "User prefers TypeScript", facts: ["prefers TypeScript"] })` |
| "we use PostgreSQL in this project" | `memory_save({ kind: "decision", title: "DB: PostgreSQL", content: "Project uses PostgreSQL", facts: ["database: PostgreSQL"] })` |
| "my name is [X]" | Save it immediately |
| "our team has [N] developers" | Save it immediately |

**Personal info = save it. Always. Without exception.**

## RULE 3 — Save after significant responses (MANDATORY)

After responding to any of these topics, call `memory_save`:

- Architectural decisions → `kind: "architecture"`
- Bug fixes and their root cause → `kind: "bug_fix"`  
- Technical decisions made → `kind: "decision"`
- Important observations about the codebase → `kind: "observation"`
- End-of-session summaries → `kind: "summary"`

**Do this automatically, EVEN if the user doesn't ask you to.**

Example — after explaining an architecture decision:
```
memory_save({
  kind: "architecture",
  title: "Auth: JWT with RS256 for stateless auth",
  content: "Decided to use JWT tokens signed with RS256 for authentication. 
            Tokens expire after 24h. Refresh tokens stored in Redis.",
  facts: ["JWT + RS256", "24h expiry", "Redis for refresh tokens"],
  concepts: ["authentication", "jwt", "redis"]
})
```

## RULE 4 — "Remember X" = save immediately

If the user says any of:
- "remember that..."
- "don't forget that..."
- "note that..."
- "keep in mind..."
- "save that..."

→ Call `memory_save` **immediately** with exactly what they said.

## RULE 5 — Never read source files for what's in memory

Before reading ANY source file, search memory first:
```
memory_search({ query: "<what you're looking for>" })
```
If the answer is in memory, use that. Only read source files if memory doesn't have it.

**Token efficiency:** reading memory costs ~10x fewer tokens than reading source files.

## RULE 6 — Private mode

If the user writes `<private>` before content, do NOT save that to memory.
Example: `<private> my password is 1234` → do not save.

## Available memory tools

| Tool | Use when |
|------|----------|
| `memory_recent({ limit: 10 })` | Start of every session |
| `memory_search({ query: "..." })` | Before answering any question |
| `memory_context({ query: "..." })` | For ready-to-inject formatted context |
| `memory_get({ ids: [...] })` | For full details on specific items |
| `memory_save({ kind, title, content, facts, concepts })` | After learning anything important |
<!-- copilot-mem:managed -->