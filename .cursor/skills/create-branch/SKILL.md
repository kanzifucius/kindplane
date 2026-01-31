---
name: create-branch
description: Create Branch and Commit Changes
disable-model-invocation: true
---

# Create Branch and Commit Changes

Analyse the uncommitted changes in the current working directory and create a well-named branch with an appropriate commit.

## Instructions

1. **Analyse uncommitted changes**: Run `git status` and `git diff` to identify all modified, added, and deleted files that are not yet committed.

2. **Understand the changes**: Read the changed files to understand the purpose and scope of the modifications.

3. **Determine a branch name**: Based on the changes, create a descriptive branch name following this format:
   - `feat/<description>` - for new features
   - `fix/<description>` - for bug fixes
   - `docs/<description>` - for documentation changes
   - `refactor/<description>` - for code refactoring
   - `chore/<description>` - for maintenance tasks
   - Use kebab-case for the description (e.g., `feat/add-user-authentication`)

4. **Create the branch**: Create and checkout the new branch with `git checkout -b <branch-name>`.

5. **Stage and commit**: Stage all changes and create a commit using conventional commits format:
   - `feat: <description>` - for new features
   - `fix: <description>` - for bug fixes
   - `docs: <description>` - for documentation
   - `refactor: <description>` - for refactoring
   - `chore: <description>` - for maintenance
   - Include a meaningful commit body if the changes warrant additional explanation.

6. **Report**: Summarise what was done, including the branch name and commit message used.
