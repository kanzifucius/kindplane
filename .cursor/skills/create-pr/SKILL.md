---
name: create-pr
description: Create GitHub Pull Request
disable-model-invocation: true
---

# Create GitHub Pull Request

Analyse the current branch, check for uncommitted changes, commit if needed using conventional commits, and create a well-structured GitHub Pull Request.

## Instructions

1. **Check for uncommitted changes**: 
   - Run `git status` to check for uncommitted changes
   - If there are uncommitted changes, proceed to step 2
   - If there are no uncommitted changes, proceed to step 3

2. **Commit uncommitted changes**:
   - Run `git diff` and `git status` to understand all changes
   - Read the changed files to understand the purpose and scope of modifications
   - Determine the commit type and scope based on conventional commits:
     - `feat(scope): <description>` - for new features
     - `fix(scope): <description>` - for bug fixes
     - `docs(scope): <description>` - for documentation changes
     - `refactor(scope): <description>` - for code refactoring
     - `test(scope): <description>` - for test additions
     - `chore(scope): <description>` - for maintenance tasks
   - Determine the scope from the changed files (e.g., `provider`, `helm`, `config`, `credentials`, `docs`)
   - **Prompt the user before committing**: 
     - Present the proposed commit message (type, scope, and description)
     - Show a summary of the changes that will be committed
     - Ask the user to confirm: "Do you want to proceed with this commit? (yes/no)"
     - Wait for user confirmation before proceeding
     - If the user declines, ask if they want to modify the commit message or skip committing
   - Only after user confirmation:
     - Stage changes safely: use `git add -u` to stage only tracked files, or run `git status` and explicitly stage specific files (e.g. `git add path/to/file`) so you do not accidentally stage sensitive or untracked files. Avoid blanket `git add .` unless the user has confirmed the full working tree.
     - Create a commit with the confirmed message following the format above
     - Include a commit body if the changes warrant additional explanation

3. **Get current branch information**:
   - Run `git branch --show-current` to get the current branch name
   - Run `git log origin/main..HEAD` (or `origin/master` if main doesn't exist) to see commits not yet on main/master
   - Determine the default branch (check `git remote show origin` or try `main` first, then `master`)

4. **Check if PR already exists**:
   - Run `gh pr list --head <branch-name>` to check if a PR already exists for this branch (use the same `<branch-name>` from step 3 throughout this step)
   - If no PR exists, proceed to step 5
   - If a PR exists:
     - **Check for unpushed commits** on the current branch: run `git rev-list --count origin/<branch-name>..HEAD` (if the command fails because `origin/<branch-name>` does not exist, treat as zero unpushed commits)
     - If the count is **zero**: inform the user and return the existing PR URL; do not push or create a new PR
     - If the count is **greater than zero** (unpushed commits exist):
       - Inform the user that a PR already exists for this branch but there are unpushed commits that would update it
       - **Prompt the user**: "There are unpushed commits on <branch-name>. Push to update the existing PR? (yes/no)"
       - If the user **accepts**: run `git push origin <branch-name>`. Do not assume success:
         - **If the push succeeds**: return the existing PR URL (as in the decline case below) and continue to the report (step 10)
         - **If the push fails** (e.g. merge conflicts, network or permission errors): inform the user of the error; provide the git error output and any instructions to resolve it (e.g. rebase, force-push policy, or credential/network checks). Still return the existing PR URL and note that the push must be resolved manually before the PR can be updated; then continue to the report (step 10)
       - If the user **declines**: do not push; return the existing PR URL and note that the PR does not yet include the unpushed commits
       - Only after the user has either consented (and push attempted) or declined, return the existing PR URL and continue to the report (step 10)

5. **Check if branch is pushed**:
   - Run `git status` to check if the branch is ahead of origin
   - If the branch is not pushed, push it with `git push -u origin <branch-name>`
   - If the branch is already pushed, proceed to step 6

6. **Analyse commits for PR content**:
   - Review all commits on the branch that are not on the default branch
   - Read the commit messages to understand the changes
   - If there are multiple commits, analyse them collectively to understand the overall change

7. **Create PR title**:
   - Based on the commit(s), create a clear, concise PR title
   - Use conventional commit format but make it more descriptive:
     - `feat(scope): Add <feature description>`
     - `fix(scope): Fix <issue description>`
     - `docs(scope): Update <documentation area>`
     - `refactor(scope): Refactor <component>`
     - `chore(scope): <maintenance task>`
   - The title should be clear and descriptive, suitable for a PR title

8. **Create PR description**:
   - Structure the description with the following sections:
   
   ## Description
   - Provide a clear overview of what this PR changes and why
   - Explain the problem it solves or the feature it adds
   
   ## Changes Made
   - List the key changes in bullet points
   - Be specific about what was modified, added, or removed
   - Group related changes together
   
   ## Testing
   - Describe how the changes were tested
   - Include any manual testing steps if applicable
   - Mention if tests were added or updated
   
   ## Related Issues
   - Link any related issues (use `Closes #<issue-number>` or `Fixes #<issue-number>` if applicable)
   - If no issues, you can omit this section
   
   ## Checklist
   - [ ] Code follows project style guidelines
   - [ ] Self-review completed
   - [ ] Comments added for complex logic
   - [ ] Documentation updated (if applicable)
   - [ ] Tests added/updated (if applicable)
   - [ ] No breaking changes (or breaking changes documented)

9. **Create the Pull Request**:
   - **Get repository information**:
     - Run `git remote get-url origin` to get the repository URL
     - Extract owner and repo name (e.g., `kanzifucius/kindplane` from `git@github.com:kanzifucius/kindplane.git`)
   
   - **First, try to use the GitHub MCP tools** to create the PR:
     - Use the repository owner and name from above
     - Set the title from step 7
     - Set the description from step 8
     - Set the base branch to `main` (or `master` if main doesn't exist)
     - Set the head branch to the current branch
     - If there are related issues, include them in the description
   
   - **If MCP tools are not available or fail**, use the GitHub CLI (`gh`) as a fallback:
     - **Verify gh CLI is authenticated**: Run `gh auth status` to confirm authentication
     - **Create temporary files**: Using the write tool, write the PR title to `/tmp/pr-title.txt` and the PR description to `/tmp/pr-body.md` so neither is interpolated into the shell command
     - **Create PR**: Run `gh pr create --title-file /tmp/pr-title.txt --body-file /tmp/pr-body.md --base <default-branch> --head <current-branch>`
     - **Handle errors**: 
       - If TLS/certificate errors occur when running `gh pr create`, request sandbox bypass for that command only: when invoking the run tool (e.g. Cursor’s terminal/sandbox executor), pass `required_permissions: ['all']` so the command runs outside the sandbox. Use only when necessary; this disables sandbox isolation for that single run.
       - If authentication fails, inform the user they need to run `gh auth login`
       - If the command fails, check error message and retry or inform user
     - **Clean up**: Always delete the temporary files (`/tmp/pr-title.txt` and `/tmp/pr-body.md`) after creating the PR (success or failure)
   
   - **Verify PR was created**:
     - The command should output a PR URL (e.g., `https://github.com/owner/repo/pull/15`)
     - If no URL is returned, check for errors and inform the user

10. **Report**:
   - Summarise what was done:
     - Whether any commits were created (if step 2 was executed)
     - The commit message(s) used (if any)
     - Current branch name
     - Number of commits included in the PR
     - The PR title and brief description summary
     - **The PR URL** (always include if PR was created successfully)
     - Any issues encountered or warnings

## Notes

- Always use British English spelling and terminology
- Ensure commit messages follow conventional commits with scope
- **IMPORTANT**: Always prompt the user and wait for confirmation before creating any commits
- The PR description should be comprehensive but concise
- If multiple commits exist, the PR should reflect the overall change, not just the latest commit
- Check for any linting or test failures before creating the PR
- Always check if a PR already exists for the branch before creating a new one; when a PR exists and there are unpushed commits (`git rev-list --count origin/<branch-name>..HEAD` > 0), prompt to push so the user can update the existing PR before returning its URL
- When using `gh` CLI, create the title and body files explicitly using the write tool (e.g. `/tmp/pr-title.txt` and `/tmp/pr-body.md`) before running the command; use `--title-file` and `--body-file` so the title and body are not interpolated into the shell
- If TLS/certificate errors occur with `gh pr create`, use the run tool with `required_permissions: ['all']` for that command only (Cursor’s sandbox bypass). Use sparingly; it disables sandbox isolation for that run.
- Always clean up temporary files (`/tmp/pr-title.txt` and `/tmp/pr-body.md`) after PR creation (success or failure)
- Verify `gh auth status` before attempting to create PR with gh CLI
- Extract repository owner/name from git remote URL for use with MCP tools
- Always include the PR URL in the final report
