---
name: context7
description: Intelligent usage of Context7 MCP for fetching current, version-specific library documentation. Automatically invoked when working with external libraries, frameworks, or APIs where up-to-date documentation improves code quality.
---

# Context7 Documentation Assistant

This skill teaches Claude when and how to effectively use the Context7 MCP server to fetch current library documentation.

## Core Principle

Context7 bridges the gap between Claude's training data (cutoff: January 2025) and the current state of fast-moving libraries. Use it proactively when library-specific knowledge matters.

## Automatic Invocation Triggers

ALWAYS use Context7 when:

1. **Library mentioned by name** in user's request
   - "implement FastAPI authentication"
   - "use psycopg2 connection pooling"
   - "Redis pub/sub with redis-py"

2. **Code involves library-specific APIs**
   - Importing third-party packages
   - Using framework-specific patterns
   - Configuring library-specific options

3. **Fast-moving ecosystems** (docs change monthly/quarterly)
   - Next.js, React, Tailwind CSS
   - FastAPI, Pydantic v2
   - Modern testing frameworks (pytest, vitest)
   - ORMs and database libraries

4. **Version-sensitive scenarios**
   - Breaking changes between major versions
   - Deprecated APIs
   - New features in recent releases

## When NOT to Use Context7

Skip Context7 for:

- **Standard library functions** (Python builtins, Node.js core modules)
- **Stable, well-known patterns** (basic SQL, HTTP concepts, Unix commands)
- **General programming concepts** (algorithms, data structures, design patterns)
- **Historical/legacy code** where training data is sufficient
- **Non-library work** (pure business logic, mathematical computations)

## Invocation Workflow

### Step 1: Detect library reference
```
User mentions: "PostgreSQL connection pooling with psycopg2"
→ Library detected: psycopg2
```

### Step 2: Resolve library ID
```
Call mcp__context7__resolve-library-id with libraryName: "psycopg2"
→ Receive Context7 ID: "psycopg/psycopg2" (or similar)
```

### Step 3: Fetch targeted documentation
```
Call mcp__context7__get-library-docs with:
- context7CompatibleLibraryID: "psycopg/psycopg2"
- topic: "connection pooling" (specific to user's need)
- tokens: 5000 (optional, defaults to 5000)
```

### Step 4: Apply documentation
Use the fetched docs to:
- Generate current, correct code
- Cite version-specific patterns
- Avoid deprecated approaches

## Token Efficiency Strategies

1. **One library at a time** - Don't bulk-fetch multiple libraries
2. **Specific topics** - Use the `topic` parameter: "FastAPI dependency injection" not just "FastAPI"
3. **Read carefully** - Context7 docs are pre-filtered and relevant
4. **Session memory** - Remember library IDs within a conversation
5. **Lazy loading** - Only fetch when about to generate library-specific code

## Error Recovery Patterns

### Library not found
```
Try variations:
- "react-query" → "tanstack-query"
- "nextjs/middleware" → "nextjs"
- "python-redis" → "redis-py"

Fallback: Training knowledge + note to user about docs availability
```

### Docs seem incomplete
```
Strategies:
1. Refine topic to be more specific
2. Try related library in ecosystem
3. Combine Context7 docs with WebSearch for very new features
```

### Rate limiting
```
Without API key: Conservative usage, batch-related queries
With API key: Normal operation
```

## Common Library Patterns

### Web Frameworks
- `fastapi` - FastAPI web framework (version-specific features)
- `flask` - Flask web framework
- `django` - Django web framework
- `express` - Express.js (Node.js)
- `nextjs` - Next.js (React framework, frequent updates)

### Frontend Libraries
- `react` - React library (hooks, concurrent features)
- `vue` - Vue.js framework
- `svelte` - Svelte framework
- `tailwindcss` - Tailwind CSS (utility classes, configuration)

### CLI & Logging
- `typer` - CLI applications (multi-command, config files)
- `click` - CLI creation
- `loguru` - Advanced logging (multi-sink, structured, rotation)

### Data & Statistical Analysis
- `pandas` - Data analysis (version 2.x nullable integers!)
- `numpy` - Numerical computing
- `scipy` - Scientific computing
- `scikit-learn` - Machine learning (API changes across versions)
- `xgboost` - Gradient boosting (version-specific parameters)

### Data Validation & Serialization
- `pydantic` - Data validation (1.x vs 2.x MAJOR breaking changes)
- `marshmallow` - Object serialization
- `zod` - TypeScript-first schema validation

### Database & Caching
- `psycopg` or `psycopg2` - PostgreSQL
- `sqlalchemy` - SQL toolkit and ORM
- `prisma` - Next-generation ORM
- `redis` or `redis-py` - Redis (queuing, caching)
- `mongodb` - MongoDB driver

### Development Tools
- `pytest` - Testing framework
- `jest` - JavaScript testing
- `vitest` - Vite-native testing framework
- `mypy` - Type checking
- `ruff` - Fast linting/formatting

### Cloud & Infrastructure
- `boto3` - AWS SDK for Python
- `google-cloud` - Google Cloud libraries
- `azure` - Azure SDK
- `terraform` - Infrastructure as code

## Integration with Other Skills

### When creating documents (docx/xlsx/pptx)
```
1. Load document skill for file format expertise
2. Use Context7 for library-specific approaches (python-docx, openpyxl)
3. Combine both for optimal results
```

### When building CLI tools
```
1. Context7 for Typer patterns (commands, config, arguments)
2. Context7 for loguru setup (sinks, formatting, rotation)
3. Your project structure and conventions
```

### When writing tests
```
1. Context7 for pytest features (fixtures, parametrize, markers)
2. Context7 for mypy type checking patterns
3. Your testing patterns for organization
```

### When debugging
```
1. Examine error messages for library clues
2. Context7 for library-specific debugging techniques
3. Stack traces often reveal version-specific issues
```

### When doing statistical analysis
```
1. Context7 for scikit-learn/xgboost version-specific APIs
2. Context7 for pandas best practices
3. Your analytical patterns and domain knowledge
```

## Best Practices

1. **Be proactive** - Don't wait to be asked; invoke when you see library names
2. **Version awareness** - Check package.json/requirements.txt first
3. **Cite sources** - Mention you're using Context7 docs briefly
4. **Verify assumptions** - Training data may be outdated for fast-moving libs
5. **Combine approaches** - Context7 + training knowledge + WebSearch as needed
6. **Use topic parameter** - Narrow down documentation to relevant sections
7. **Token management** - Adjust tokens parameter based on complexity (default: 5000)

## Example Decision Trees

### User: "Set up PostgreSQL connection pooling"
```
Decision: Use Context7
Reason: PostgreSQL drivers have version-specific pooling APIs
Steps:
1. Detect library: psycopg2 or psycopg3 (check user's environment)
2. Call mcp__context7__resolve-library-id with "psycopg2"
3. Call mcp__context7__get-library-docs with topic "connection pooling"
4. Generate current best-practice code
```

### User: "Explain bubble sort algorithm"
```
Decision: Skip Context7
Reason: Algorithmic concept, no library involved
Action: Use training knowledge directly
```

### User: "Create a Typer CLI that loads config from YAML and has multiple commands"
```
Decision: Use Context7
Reason: Typer has specific patterns for multi-command + config
Steps:
1. Call mcp__context7__resolve-library-id with "typer"
2. Call mcp__context7__get-library-docs with topic "command groups and configuration"
3. Generate current Typer patterns
```

### User: "Train an XGBoost model with cross-validation"
```
Decision: Use Context7
Reason: XGBoost parameters and APIs change across versions
Steps:
1. Check if requirements.txt exists (version awareness)
2. Call mcp__context7__resolve-library-id with "xgboost"
3. Call mcp__context7__get-library-docs with topic "cross-validation"
4. Generate version-appropriate code
```

### User: "Set up loguru with multiple log files for different modules"
```
Decision: Use Context7
Reason: loguru has specific multi-sink configuration patterns
Steps:
1. Call mcp__context7__resolve-library-id with "loguru"
2. Call mcp__context7__get-library-docs with topic "multi-sink setup"
3. Generate proper sink configuration
```

### User: "Build a Next.js app with App Router and server actions"
```
Decision: Use Context7
Reason: Next.js App Router is recent and has specific patterns
Steps:
1. Call mcp__context7__resolve-library-id with "nextjs"
2. Call mcp__context7__get-library-docs with topic "App Router server actions"
3. Generate current Next.js patterns
```

## Monitoring Your Usage

After using Context7, briefly note:
- ✓ Library ID resolved successfully
- ✓ Docs were relevant and current
- ⚠ Fell back to training knowledge (docs unavailable)
- ⚠ Combined Context7 + WebSearch (very new features)

This helps users understand your information sources.

## Special Considerations for Data Analysis Workflows

### Statistical analysis and ML pipelines
- Use Context7 for: pandas, numpy, scipy, scikit-learn, xgboost specifics
- Use training for: statistical concepts, mathematical theory
- Check versions especially for breaking changes (pandas 1.x → 2.x, pydantic 1.x → 2.x)
- Version-specific APIs matter for scikit-learn estimators and xgboost parameters

### CLI applications with Typer
- Multi-command structures: Command groups, shared options
- Configuration handling: YAML/TOML loading patterns
- Use Context7 for: Typer-specific patterns, not generic argparse

### Logging with loguru
- Multi-sink setups for different modules/projects
- Structured logging for data pipelines
- Rotation and retention policies
- Use Context7 for: loguru-specific configuration, not generic logging

### Database-heavy work
- PostgreSQL extensions and features change frequently
- Use Context7 for: psycopg versions, connection patterns, COPY operations
- Critical for: JSON operations, array handling, performance features
- Bulk loading patterns (COPY command) for large datasets

### Data validation with pydantic
- CRITICAL: pydantic 1.x vs 2.x has massive breaking changes
- Always check version before generating pydantic code
- Use Context7 for: version-specific validation patterns, field definitions
- Migration patterns if upgrading from v1 to v2

### Development tooling
- pytest: fixtures, parametrize, markers change over versions
- mypy: type checking patterns evolve
- ruff: configuration and rule sets update frequently
- Use Context7 for: current best practices, not outdated patterns

### Modern Frontend Development
- React: hooks, concurrent features, server components (Next.js)
- Next.js: App Router vs Pages Router, server actions, middleware
- Tailwind CSS: utility classes, configuration, plugins
- Use Context7 for: framework-specific patterns and best practices

## Tool Reference

### mcp__context7__resolve-library-id
**Purpose**: Resolves a package/product name to a Context7-compatible library ID

**Parameters**:
- `libraryName` (required): Library name to search for (e.g., "psycopg2", "nextjs", "react")

**Returns**: List of matching libraries with IDs in format `/org/project` or `/org/project/version`

### mcp__context7__get-library-docs
**Purpose**: Fetches up-to-date documentation for a library

**Parameters**:
- `context7CompatibleLibraryID` (required): Exact library ID from resolve-library-id
- `topic` (optional): Topic to focus documentation on (e.g., "hooks", "routing")
- `tokens` (optional): Maximum tokens to retrieve (default: 5000)

**Returns**: Relevant, current documentation for the specified library and topic

## Meta: When to Load This Skill

This skill loads automatically when:
- User mentions library/framework names
- Code generation involves third-party packages
- Documentation quality affects success

Token cost: ~50 tokens until loaded, ~2500 tokens when active
