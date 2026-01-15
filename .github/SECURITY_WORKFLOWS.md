# Security Workflows

This repository includes automated security scanning workflows to ensure code quality and security.

## Workflows

### 1. CodeQL Security Scan (`codeql-security.yml`)

**Purpose:** Comprehensive security analysis using GitHub's CodeQL engine.

**When it runs:**
- On every push to `main`/`master` branches
- On every pull request to `main`/`master` branches
- Weekly on Mondays at midnight UTC (scheduled scan)

**What it does:**
- Scans Go code for security vulnerabilities
- Checks for common security issues (SQL injection, XSS, etc.)
- Detects GraphQL-related security problems if GraphQL code is present
- Uses extended security query suite for comprehensive coverage

**Results:** Security alerts appear in the "Security" tab of the repository.

### 2. GraphQL Security Check (`graphql-security.yml`)

**Purpose:** Specialized security validation for GraphQL schemas and queries.

**When it runs:**
- When `.graphql` or `.gql` files are added or modified
- Can be manually triggered via workflow dispatch

**What it does:**
- Detects GraphQL schema files in the repository
- Validates GraphQL schema syntax
- Checks for security best practices:
  - Introspection configuration in production
  - Authentication directive usage
  - Rate limiting configurations
  - Depth limiting to prevent DoS attacks

**Note:** This workflow gracefully skips checks if no GraphQL files are present, making it future-proof for when GraphQL is added to the project.

## Security Best Practices

### For GraphQL Security:

1. **Disable Introspection in Production:** Introspection should be disabled in production environments to prevent attackers from discovering your schema.

2. **Implement Query Depth Limiting:** Prevent deeply nested queries that can cause DoS attacks.

3. **Add Rate Limiting:** Protect your GraphQL endpoint from abuse with proper rate limiting.

4. **Use Authentication Directives:** Properly secure your GraphQL fields and types with authentication and authorization directives.

5. **Validate Input:** Always validate and sanitize user input in resolvers.

## Viewing Security Results

1. Navigate to the "Security" tab in the GitHub repository
2. Click on "Code scanning alerts" to see CodeQL findings
3. Review and triage any security issues found

## Manual Workflow Execution

To manually run the GraphQL security check:
1. Go to "Actions" tab
2. Select "GraphQL Security Check" workflow
3. Click "Run workflow"

## Customization

### Adding Custom Security Checks

To add custom GraphQL security rules, edit `.github/workflows/graphql-security.yml` and add checks in the "GraphQL Security Best Practices Check" step.

### Adjusting CodeQL Queries

To modify which CodeQL queries run, edit the `queries` parameter in `.github/workflows/codeql-security.yml`:

```yaml
queries: security-extended,security-and-quality
```

Available query suites:
- `security-extended` - Extended security queries
- `security-and-quality` - Security and code quality queries
- Custom query packs can also be specified

## Troubleshooting

### Workflow Fails on Autobuild

If the CodeQL autobuild step fails, you may need to add custom build commands. Replace the `autobuild` step with:

```yaml
- name: Build
  run: |
    make build
```

### GraphQL Tools Installation Fails

If Node.js-based GraphQL tools fail to install, check the Node.js version in the workflow and ensure it matches your requirements.

## Contributing

When contributing code:
1. Ensure all security workflows pass
2. Address any security alerts before merging
3. Add appropriate security tests for new features
