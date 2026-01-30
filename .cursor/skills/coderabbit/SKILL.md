---
name: coderabbit
description: Run CodeRabbit CLI for AI code reviews and integrate findings into the development workflow. Use when the user wants a code review, to run CodeRabbit after changes, or to fix issues CodeRabbit reported.
---

# CodeRabbit Code Review

Use the CodeRabbit CLI to review code from within Cursor. CodeRabbit is already installed; run it as a command-line tool to get AI-powered feedback on uncommitted or committed changes.

## When to Use

- User asks to "run CodeRabbit", "review my code", or "run a code review"
- After implementing a feature and the user wants issues caught before commit/PR
- User asks to "fix CodeRabbit issues" or "address CodeRabbit suggestions"

## Prerequisites

- CodeRabbit CLI installed (install: `curl -fsSL https://cli.coderabbit.ai/install.sh | sh`)
- Authenticated: run `coderabbit auth status`; if not logged in, run `coderabbit auth login`

## How to Run CodeRabbit

1. **Review uncommitted changes (most common)**  
   Use this when reviewing current working-directory changes:
   ```bash
   coderabbit --prompt-only -t uncommitted
   ```
   `--prompt-only` gives Cursor concise, token-efficient output (file, line, severity, suggestion).

2. **Review committed changes**  
   ```bash
   coderabbit --prompt-only -t committed
   ```

3. **Review all changes (committed + uncommitted)**  
   ```bash
   coderabbit --prompt-only -t all
   ```

4. **Different base branch**  
   If the default branch is not `main`:
   ```bash
   coderabbit --prompt-only -t uncommitted --base develop
   ```

5. **Help**  
   ```bash
   cr -h
   ```
   or `coderabbit -h` for full CLI options.

## Limits

- **Do not run CodeRabbit more than 3 times** for the same set of changes in one session. After that, summarise remaining issues and stop.
- Reviews can take several minutes (about 7–30 depending on scope). Run in the background or warn the user if the run will be long.

## After CodeRabbit Finishes

1. Read the `--prompt-only` output (plain text with file locations, severity, and suggestions).
2. Build a short task list of issues to fix (by file and severity).
3. Apply fixes for each finding; prefer fixing critical/high first, then nits if the user asked for "all" or "fix everything".
4. If the user said "fix critical only" or "ignore nits", only address critical/high findings.

## Project Configuration

This repo has a [.coderabbit.yaml](.coderabbit.yaml) that sets:

- **Language**: British English
- **Profile**: chill (lighter feedback, fewer nits)
- **Tone**: Concise, direct, actionable; British English spelling

Respect that style when describing or applying CodeRabbit feedback (e.g. "colour" not "color").

## Optional: Cursor Rule

To make Cursor always use CodeRabbit correctly, add a rule via Cursor’s `@rule` and paste:

```markdown
# Running the CodeRabbit CLI
CodeRabbit is already installed in the terminal. Run it to review code.
- Help: `cr -h` or `coderabbit -h`
- Prefer: `coderabbit --prompt-only -t uncommitted` for uncommitted changes
- Do not run CodeRabbit more than 3 times for the same set of changes
```

## References

- [CodeRabbit Cursor integration](https://docs.coderabbit.ai/cli/cursor-integration)
- [CodeRabbit configuration](https://docs.coderabbit.ai/configuration)
