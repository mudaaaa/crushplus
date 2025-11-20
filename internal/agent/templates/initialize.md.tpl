Analyze this codebase and create/update **{{.Config.Options.InitializeAs}}** to serve as the authoritative knowledge base for future agents working in this repository.

**First**: Check if the directory is empty or contains only configuration files. If so, stop immediately and respond: "Directory appears empty or only contains config. Add source code first, then run this command to generate {{.Config.Options.InitializeAs}}."

**Goal**: Create a comprehensive operational guide that enables any agent to immediately understand, build, test, and modify this codebase without prior context.

**Discovery Process**:

1.  **Inventory**: Run `ls -R` (or equivalent) to map the full project structure.
2.  **Configuration Analysis**: Identify project type and dependencies by reading files like `package.json`, `go.mod`, `Cargo.toml`, `pom.xml`, `requirements.txt`, `Dockerfile`, `docker-compose.yml`, etc.
3.  **Existing Context**: Check for and read existing documentation or rule files (`.cursor/rules/*.md`, `.cursorrules`, `.github/copilot-instructions.md`, `claude.md`, `agents.md`, `README.md`, `CONTRIBUTING.md`).
4.  **Command Extraction**: Locate build, test, lint, and run commands in `Makefile`, `Taskfile`, `package.json` scripts, CI/CD configurations (e.g., `.github/workflows`), or shell scripts.
5.  **Pattern Recognition**: Read a diverse sample of source files to identify:
    *   Architectural patterns (e.g., MVC, Hexagonal, Clean Architecture).
    *   Coding style and conventions (naming, formatting, error handling).
    *   Testing patterns (unit vs integration, libraries used).

**Content Requirements for {{.Config.Options.InitializeAs}}**:

*   **Project Overview**: Brief summary of what the project does and its primary tech stack (languages, frameworks, key libraries).
*   **Operational Commands**: Verified commands for:
    *   **Setup/Install**: How to install dependencies.
    *   **Development**: How to start the local dev server or watcher.
    *   **Testing**: How to run the full suite and specific tests.
    *   **Building**: How to build for production.
    *   **Linting/Formatting**: How to enforce code style.
*   **Architecture & Structure**:
    *   Key directories and their specific responsibilities.
    *   Core design patterns used throughout the codebase.
    *   Data flow summary (if discernible).
*   **Coding Standards**:
    *   Naming conventions (files, variables, functions).
    *   Preferred idioms and patterns.
    *   Error handling approach.
*   **Gotchas & Edge Cases**: Specific quirks, non-standard implementations, or critical configuration details observed.

**Format**: Use clear Markdown with headers, code blocks for commands, and bullet points for readability. Structure it logically for quick lookup.

**Critical Rules**:
*   **Evidence-Based**: Only document what you verify exists. Do not hallucinate commands or patterns.
*   **Specificity**: Prefer specific commands (e.g., `npm run test:unit`) over generic advice.
*   **Completeness**: If a standard workflow (like testing) is missing, note that it was not found rather than inventing one.
