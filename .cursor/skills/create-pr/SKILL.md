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
     - Stage all changes with `git add .`
     - Create a commit with the confirmed message following the format above
     - Include a commit body if the changes warrant additional explanation

3. **Get current branch information**:
   - Run `git branch --show-current` to get the current branch name
   - Run `git log origin/main..HEAD` (or `origin/master` if main doesn't exist) to see commits not yet on main/master
   - Determine the default branch (check `git remote show origin` or try `main` first, then `master`)

4. **Check if branch is pushed**:
   - Run `git status` to check if the branch is ahead of origin
   - If the branch is not pushed, push it with `git push -u origin <branch-name>`
   - If the branch is already pushed, proceed to step 5

5. **Analyse commits for PR content**:
   - Review all commits on the branch that are not on the default branch
   - Read the commit messages to understand the changes
   - If there are multiple commits, analyse them collectively to understand the overall change

6. **Create PR title**:
   - Based on the commit(s), create a clear, concise PR title
   - Use conventional commit format but make it more descriptive:
     - `feat(scope): Add <feature description>`
     - `fix(scope): Fix <issue description>`
     - `docs(scope): Update <documentation area>`
     - `refactor(scope): Refactor <component>`
     - `chore(scope): <maintenance task>`
   - The title should be clear and descriptive, suitable for a PR title

7. **Create PR description**:
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

8. **Create the Pull Request**:
   - First, try to use the GitHub MCP tools to create the PR:
     - Set the title from step 6
     - Set the description from step 7
     - Set the base branch to `main` (or `master` if main doesn't exist)
     - Set the head branch to the current branch
     - If there are related issues, include them in the description
   - **If MCP tools are not available**, use the GitHub CLI (`gh`) as a fallback:
     - Save the PR description to a temporary file (e.g., `/tmp/pr-description.md`)
     - Run `gh pr create --title "<title from step 6>" --body-file /tmp/pr-description.md --base <default-branch> --head <current-branch>`
     - If there are related issues, add `--body` parameter with the description including issue references, or use `--body-file` and include issues in the file
     - Clean up the temporary file after creating the PR
   - Verify that `gh` CLI is installed and authenticated if using the fallback method

9. **Report**:
   - Summarise what was done:
     - Whether any commits were created
     - The commit message(s) used
     - The PR title and description
     - The PR URL (if available)

## Notes

- Always use British English spelling and terminology
- Ensure commit messages follow conventional commits with scope
- **IMPORTANT**: Always prompt the user and wait for confirmation before creating any commits
- The PR description should be comprehensive but concise
- If multiple commits exist, the PR should reflect the overall change, not just the latest commit
- Check for any linting or test failures before creating the PR
- If GitHub MCP tools are not available, fall back to using the GitHub CLI (`gh pr create`)
- Ensure the GitHub CLI is installed and authenticated if using the fallback method
