---
name: ard-skill-creator
description: Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends agents capabilities with specialized knowledge, workflows, or tool integrations.
version: 1.2.0
author: OpenAI
---

# Skill creator

This skill provides guidance for creating effective skills.

## About Skills
Skills are modular, self-contained folders that extend agent capabilities by providing specialized knowledge, workflows, and tools. Think of them as "onboarding guides" for specific domains or tasks—they transform agent from a general-purpose into a specialized agent equipped with procedural knowledge that no model can fully possess.

Make it a Skill when:

- The capability can be expressed as instructions + shell commands + existing tools
- It wraps an external CLI or API that the agent can call via terminal or web_extract
- It doesn't need custom Python integration or API key management baked into the agent
- Examples: arXiv search, git workflows, Docker management, PDF processing, email via CLI tools

# Skill Directory Structure

Skills live in skills/ organized in flat list:
- `/{agnent}/skills` - per agent private
- `/skills` - shared skills (available for all agents)

```
skills/
└── skill-name/
    ├── SKILL.md (required)
    ├── scripts/          - Executable code (Python/Bash/etc.) (optional)
    ├── references/       - Documentation intended to be loaded into context as needed (optional)
    └── assets/           - Files used in output (templates, icons, fonts, etc.) (optional)
```

# SKILL.md Format

**Frontmatter (YAML)**: Contains name and description fields. It is very important to be clear and comprehensive in describing what the skill is, and when it should be used.

**Body (Markdown)**: Instructions and guidance for using the skill. Only loaded AFTER the skill triggers (if at all).

```markdown
---
name: my-skill
description: Brief description (shown in skill search results)
version: 1.0.0
author: Your Name
required_environment_variables:
  - name: TENOR_API_KEY
    prompt: Tenor API key
    help: Get a key from https://developers.google.com/tenor
    required_for: full functionality
---

# Skill Title

Brief intro.

## When to Use
Trigger conditions — when should the agent load this skill?

## Quick Reference
Table of common commands or API calls.

## Procedure
Step-by-step instructions the agent follows.

## Pitfalls
Known failure modes and how to handle them.

## Verification
How the agent confirms it worked.
```
# Secure Setup on Load

Use required_environment_variables when a skill needs an API key or token.
It makes a hint to the agent for env is requred.

```yaml
required_environment_variables:
  - name: TENOR_API_KEY
    prompt: Tenor API key
    help: Get a key from https://developers.google.com/tenor
    required_for: full functionality
```

Your skill's scripts can access $TENOR_API_KEY (or os.environ["TENOR_API_KEY"] in Python)
Only user can add new env's manually, when new skill need a env. 
You should make request to user, for add a new env.

# Skill Guidelines

## Skill Naming

- Use lowercase letters, digits, and hyphens only; normalize user-provided titles to hyphen-case (e.g., "Plan Mode" -> plan-mode).
- When generating names, generate a name under 64 characters (letters, digits, hyphens).
- Prefer short, verb-led phrases that describe the action.
- Namespace by tool when it improves clarity or triggering (e.g., gh-address-comments, linear-address-issue).
- Name the skill folder exactly after the skill name.

## Concise is Key

The context window is a public good. Skills share the context window with everything else agent needs: system prompt, conversation history, other Skills' metadata, and the actual user request.

Default assumption: agent is already very smart. Only add context agent doesn't already have. Challenge each piece of information: "Does agent really need this explanation?" and "Does this paragraph justify its token cost?"

Prefer concise examples over verbose explanations.

## Set Appropriate Degrees of Freedom

Match the level of specificity to the task's fragility and variability:

**High freedom (text-based instructions)**: Use when multiple approaches are valid, decisions depend on context, or heuristics guide the approach.

**Medium freedom (pseudocode or scripts with parameters)**: Use when a preferred pattern exists, some variation is acceptable, or configuration affects behavior.

**Low freedom (specific scripts, few parameters)**: Use when operations are fragile and error-prone, consistency is critical, or a specific sequence must be followed.

## No External Dependencies

Prefer stdlib Python, curl, and existing Hermes tools (web_extract, terminal, read_file). If a dependency is needed, document installation steps in the skill.
Progressive Disclosure

## Progressive Disclosure

Put the most common workflow first. Edge cases and advanced usage go at the bottom. This keeps token usage low for common tasks.
Include Helper Scripts

## Include Helper Scripts

Executable code (Python/Bash/etc.) for tasks that require deterministic reliability or are repeatedly rewritten.
- **When to include**: When the same code is being rewritten repeatedly or deterministic reliability is needed
- **Example**: scripts/rotate_pdf.py for PDF rotation tasks

For XML/JSON parsing or complex logic, include helper scripts in scripts/ — don't expect the LLM to write parsers inline every time.

So a SKILL.md can tell the agent to run a bundled script directly with:

```markdown
To analyse the input, run:
`node ${HERMES_SKILL_DIR}/scripts/analyse.js <input>`
```

The agent sees the substituted absolute path and invokes the shell tool with a ready-to-run command — no path math, no extra skill_view round-trip.

## References (references/)

Documentation and reference material intended to be loaded as needed into context to inform Codex's process and thinking.
- **When to include**: For documentation that Codex should reference while working
- **Examples**: references/finance.md for financial schemas, references/mnda.md for company NDA template, references/policies.md for company policies, references/api_docs.md for API specifications

## Assets (assets/)

Files not intended to be loaded into context, but rather used within the output Codex produces.
- **When to include**: When the skill needs files that will be used in the final output
- **Examples**: assets/logo.png for brand assets, assets/slides.pptx for PowerPoint templates, assets/frontend-template/ for HTML/React boilerplate, assets/font.ttf for typography

## What to Not Include in a Skill

A skill should only contain **essential files** that directly support its **functionality**. Do NOT create extraneous documentation or auxiliary files, including:
- README.md
- INSTALLATION_GUIDE.md
- QUICK_REFERENCE.md
- CHANGELOG.md
- etc...

The skill should only contain the information needed for an AI agent to do the job at hand. It should not contain auxiliary context about the process that went into creating it, setup and testing procedures, user-facing documentation, etc. Creating additional documentation files just adds clutter and confusion.

## Progressive Disclosure Patterns

Keep SKILL.md body to the essentials and under 500 lines to minimize context bloat. Split content into separate files when approaching this limit. When splitting out content into other files, it is very important to reference them from SKILL.md and describe clearly when to read them, to ensure the reader of the skill knows they exist and when to use them.

Key principle: When a skill supports multiple variations, frameworks, or options, keep only the core workflow and selection guidance in SKILL.md. Move variant-specific details (patterns, examples, configuration) into separate reference files.

### Pattern 1: High-level guide with references

In skill example:

```markdown
# PDF Processing

## Quick start

Extract text with pdfplumber:
[code example]

## Advanced features

- **Form filling**: See [FORMS.md](FORMS.md) for complete guide
- **API reference**: See [REFERENCE.md](REFERENCE.md) for all methods
- **Examples**: See [EXAMPLES.md](EXAMPLES.md) for common patterns
```

Agent loads FORMS.md, REFERENCE.md, or EXAMPLES.md only when needed.

### Pattern 2: Domain-specific organization

For Skills with multiple domains, organize content by domain to avoid loading irrelevant context:
```
bigquery-skill/
├── SKILL.md (overview and navigation)
└── reference/
    ├── finance.md (revenue, billing metrics)
    ├── sales.md (opportunities, pipeline)
    ├── product.md (API usage, features)
    └── marketing.md (campaigns, attribution)
```
When a user asks about sales metrics, Agent only reads sales.md.

Similarly, for skills supporting multiple frameworks or variants, organize by variant:
```
cloud-deploy/
├── SKILL.md (workflow + provider selection)
└── references/
    ├── aws.md (AWS deployment patterns)
    ├── gcp.md (GCP deployment patterns)
    └── azure.md (Azure deployment patterns)
```
When the user chooses AWS, Codex only reads aws.md.

### Pattern 3: Conditional details

Show basic content, link to advanced content:
```markdown
# DOCX Processing

## Creating documents

Use docx-js for new documents. See [DOCX-JS.md](DOCX-JS.md).

## Editing documents

For simple edits, modify the XML directly.

**For tracked changes**: See [REDLINING.md](REDLINING.md)
**For OOXML details**: See [OOXML.md](OOXML.md)
```

Agent reads REDLINING.md or OOXML.md only when the user needs those features.

## Important:
- Avoid deeply nested references - Keep references one level deep from SKILL.md. All reference files should link directly from SKILL.md.
- Structure longer reference files - For files longer than 100 lines, include a table of contents at the top so Agent can see the full scope when previewing.

# Workflow

Skill creation involves these steps:
1. Understand the skill with concrete examples
2. Plan reusable skill contents (scripts, references, assets)
3. Initialize the skill (run init_skill.py)
4. Edit the skill (implement resources and write SKILL.md)
5. Validate the skill (run quick_validate.py)
6. Iterate based on real usage

Follow these steps in order, skipping only if there is a clear reason why they are not applicable.

## Step 1: Understanding the Skill with Concrete Examples

Skip this step only when the skill's usage patterns are already clearly understood. It remains valuable even when working with an existing skill.

To create an effective skill, clearly understand concrete examples of how the skill will be used. This understanding can come from either direct user examples or generated examples that are validated with user feedback.

For example, when building an image-editor skill, relevant questions include:
- "What functionality should the image-editor skill support? Editing, rotating, anything else?"
- "Can you give some examples of how this skill would be used?"
- "I can imagine users asking for things like 'Remove the red-eye from this image' or 'Rotate this image'. Are there other ways you imagine this skill being used?"
- "What would a user say that should trigger this skill?"

To avoid overwhelming users, avoid asking too many questions in a single message. Start with the most important questions and follow up as needed for better effectiveness.

Conclude this step when there is a clear sense of the functionality the skill should support.

## Step 2: Planning the Reusable Skill Contents

To turn concrete examples into an effective skill, analyze each example by:

1. Considering how to execute on the example from scratch
2. Identifying what scripts, references, and assets would be helpful when executing these workflows repeatedly

Example: When building a `pdf-editor` skill to handle queries like "Help me rotate this PDF," the analysis shows:

1. Rotating a PDF requires re-writing the same code each time
2. A `scripts/rotate_pdf.py` script would be helpful to store in the skill

Example: When designing a `frontend-webapp-builder` skill for queries like "Build me a todo app" or "Build me a dashboard to track my steps," the analysis shows:

1. Writing a frontend webapp requires the same boilerplate HTML/React each time
2. An `assets/hello-world/` template containing the boilerplate HTML/React project files would be helpful to store in the skill

Example: When building a `big-query` skill to handle queries like "How many users have logged in today?" the analysis shows:

1. Querying BigQuery requires re-discovering the table schemas and relationships each time
2. A `references/schema.md` file documenting the table schemas would be helpful to store in the skill

To establish the skill's contents, analyze each concrete example to create a list of the reusable resources to include: scripts, references, and assets.

## Step 3: Initializing the Skill

At this point, it is time to actually create the skill.

Skip this step only if the skill being developed already exists. In this case, continue to the next step.

When creating a new skill from scratch, always run the init_skill.py script. The script conveniently generates a new template skill directory that automatically includes everything a skill requires, making the skill creation process much more efficient and reliable.

Usage:

`scripts/init_skill.py <skill-name> --path <output-directory> [--resources scripts,references,assets] [--examples]`

Examples:
```shell
scripts/init_skill.py my-skill --path skills/public
scripts/init_skill.py my-skill --path skills/public --resources scripts,references
scripts/init_skill.py my-skill --path skills/public --resources scripts --examples
```

The script:
- Creates the skill directory at the specified path
- Generates a SKILL.md template with proper frontmatter and TODO placeholders
- Optionally creates resource directories based on --resources
- Optionally adds example files when --examples is set

After initialization, customize the SKILL.md and add resources as needed. If you used --examples, replace or delete placeholder files.

Generate display_name, short_description, and default_prompt by reading the skill, then pass them as --interface key=value to init_skill.py or regenerate with:

scripts/generate_openai_yaml.py <path/to/skill-folder> --interface key=value
> ensure you has access to path

## Step 4: Edit the Skill

When editing the (newly-generated or existing) skill, remember that the skill is being created for another instance of Agent to use. Include information that would be beneficial and non-obvious to Agent. Consider what procedural knowledge, domain-specific details, or reusable assets would help another Agent instance execute these tasks more effectively.

### Start with Reusable Skill Contents

To begin implementation, start with the reusable resources identified above: scripts/, references/, and assets/ files. Note that this step may require user input. For example, when implementing a brand-guidelines skill, the user may need to provide brand assets or templates to store in assets/, or documentation to store in references/.

Added scripts must be tested by actually running them to ensure there are no bugs and that the output matches what is expected. If there are many similar scripts, only a representative sample needs to be tested to ensure confidence that they all work while balancing time to completion.

If you used --examples, delete any placeholder files that are not needed for the skill. Only create resource directories that are actually required.

### Update SKILL.md

Writing Guidelines: Always use imperative/infinitive form.

**Frontmatter**
Write the YAML frontmatter with name and description:
```yaml
    name: The skill name
    description: This is the primary triggering mechanism for your skill, and helps Agent understand when to use the skill.
```

**Body**

Write instructions for using the skill and its bundled resources.

## Step 5: Validate the Skill

Once development of the skill is complete, validate the skill folder to catch basic issues early:
```shell
scripts/quick_validate.py <path/to/skill-folder>
```
The validation script checks YAML frontmatter format, required fields, and naming rules. If validation fails, fix the reported issues and run the command again.

## Step 6: Iterate

After testing the skill, users may request improvements. Often this happens right after using the skill, with fresh context of how the skill performed.

Iteration workflow:
- Use the skill on real tasks
- Notice struggles or inefficiencies
- Identify how SKILL.md or bundled resources should be updated
- Implement changes and test again

Run the skill on sub agent and verify the agent follows the instructions correctly:

# Delivery

Skill is owned by domain specialist

If your skill is domain-specific (useful for a particular task, e.g. code or web search) but not universally needed (e.g. a paid service integration, a heavy dependency), it belongs to that domain.

If the skill belongs to your domain, keep it in your private skills folder: `/<your_folder>/skills`.

The skill is maintained by whoever owns it. If the skill is meant for a specific agent, deliver it to that agent:
- Move the skill folder to `/shared`
- Call that agent as a sub-agent and tell it to move `/shared/new-skill` into its own private skills folder

when your skill is for public usage (e.g. how to communicate with human) put it into `/shared`
