---
description: Automatic commit message generator and fast AI-powered commit for all current changes
---

// turbo-all

This workflow automatically stages all changes, generates a descriptive commit message, and commits them in one go.

### Steps:

1. **Audit Packages**: Run `make audit` to check for package vulnerabilities.
   ```bash
   make audit
   ```
2. **Build Check**: Check if the project is buildable (compilation check only).
   ```bash
   go build ./...
   ```
3. **Stage All Changes**: Automatically stage all modified and new files.
   ```bash
   git add .
   ```
4. **Analyze Changes**: Get the diff of staged changes to understand the context.
   ```bash
   git diff --cached
   ```
5. **Generate & Commit**: Generate a professional message following [Conventional Commits](https://www.conventionalcommits.org/) and execute the commit.
   ```bash
   git commit -m "<ai_generated_message>"
   ```
6. **Push**: Optionally push the changes.
   ```bash
   git push
   ```
7. **Prepare PR Description**: If a PR will be created, generate and print a PR description that can be pasted into the PR body.
   - Use a concise `Summary` section with the overall intent of the PR.
   - Use a `Changes` section for the concrete implementation details.
   - Use a `Verification` section with checked commands.
   - Include `Commits` when the PR is intentionally split into meaningful commits.
   - Include `Review Notes` only for compatibility notes, reviewer focus areas, migrations, or release concerns.
   - Keep each bullet focused and avoid long narrative paragraphs.
   ```markdown
   ## Summary

   <One short paragraph explaining the purpose of the PR.>

   ## Changes

   - ...

   ## Commits

   - `<commit message>`

   ## Verification

   - [x] `...`

   ## Review Notes

   - ...
   ```
