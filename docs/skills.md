# Skills

A **skill** is a package of instructions that teaches an agent how to do
something specific — a procedure, a workflow, background knowledge. Skills are
plain markdown files, so they are easy to write, version and share.

## Skill anatomy

A skill is a folder containing a `SKILL.md` file. The folder name is arbitrary;
folders may be nested. The file has a small frontmatter block and the actual
skill text below:

```markdown
---
name: code-review
description: Review a pull request: check diff, run tests, report findings
allowed-tools:
  - shell
  - filesystem
---

# Code review

1. Check out the branch.
2. Run the test suite and note failures.
3. Read the diff and look for the problems listed in ...
```

Frontmatter fields:

| Field | Meaning |
|-------|---------|
| `name` | Short skill name |
| `description` | One or two sentences describing what the skill does. This is what the agent sees in its skill index — make it informative |

The body of the file is the skill itself: a step-by-step instruction the agent
follows when it uses the skill.

## Where skills live

There are two places for skills in the data directory
([configuration.md](configuration.md)):

- **Shared skills** — folder `skills/` at the top level. Visible to **all**
  agents.
- **Private skills** — folder `skills/` inside an agent's own directory
  (`<agent_id>/skills/`). Visible only to that agent.

Both are combined for the agent, so a shared skill plus an agent's private
skills are all available. If the same skill exists in both places, the shared
copy takes precedence.

## How skills reach the agent

When a session starts, the agent's system message includes a **skill index** —
the list of available skills with their paths and descriptions. The agent sees
something like:

```
- (skills/code-review/SKILL.md) Review a pull request: ...
```

To actually use a skill, the agent opens its `SKILL.md` file with the
filesystem tools and follows the instructions. Skills are picked up without a
restart — create a `SKILL.md`, and it appears in new sessions. The system
message is cached for a while, so a skill created in the middle of an ongoing
session shows up after the cache expires or in a new session.

> Make sure the agent that should use a skill has the `filesystem` tool server
> attached (see [tools.md](tools.md)) — otherwise it can see the skill index
> but cannot read the files.
