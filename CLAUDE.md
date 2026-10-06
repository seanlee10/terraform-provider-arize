# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Terraform provider for Arize, enabling Infrastructure as Code management of Arize resources (users, roles, API keys, spaces, and space memberships). It uses the Terraform Plugin Framework (Go-based) and communicates with Arize's GraphQL API.

## Architecture

### High-Level Design

```
User Terraform Config (.tf files)
         ↓
Terraform Plugin Framework
         ↓
Provider (provider.go)
  ├── Resources (5 total)
  ├── Data Sources (3 total)
  └── Configuration (API key, endpoint)
         ↓
GraphQL Client (internal/client/)
  ├── Authentication (x-api-key header)
  ├── Queries (read operations)
  └── Mutations (create/update/delete)
         ↓
Arize GraphQL API (app.arize.com/graphql)
```

### Core Components

**Provider & Resources** (`internal/provider/`)
- `provider.go` - Main provider configuration, registers resources and data sources
- `resource_*.go` - CRUD operations for: user, role, api_key, space, space_member
- `data_source_*.go` - Read-only data sources for: users, roles, api_keys

**GraphQL Client** (`internal/client/`)
- `graphql.go` - HTTP client with x-api-key authentication
- `queries.go` - GraphQL read operations (GetUser, ListUsers, etc.)
- `mutations.go` - GraphQL write operations (CreateUser, AssignSpaceMembership, etc.)
- `models.go` - Data structures for users, roles, API keys
- `models_space.go` - Data structures for spaces and space members

### Critical API Knowledge

**Authentication Header**
Arize uses `x-api-key` header, NOT `Authorization: Bearer`. This is different from many GraphQL APIs and is easy to get wrong.

**Query Patterns**
- **List resources**: Via `account.users`, `account.roles`, etc. (not root-level queries)
- **Get single resource**: Via `node(id)` query with GraphQL fragments (e.g., `... on User { ... }`)
- **Create/update**: Via mutations with `Input` types

**Example Query**
```graphql
query GetUser($id: ID!) {
  node(id: $id) {
    __typename
    ... on User {
      id
      email
      name
      status
      userType
      createdAt
    }
  }
}
```

### Space Roles

Valid space roles: `admin`, `member`, `readOnly`, `annotator`. These are case-sensitive enum values used in mutations.

### Known Limitations

1. **Organization ID Hardcoded** - Space creation requires an organization ID, currently hardcoded in `resource_space.go:getDefaultOrganizationID()`. Should be retrieved dynamically from the account.
2. **Space Updates Not Implemented** - `updateSpace` mutation exists but isn't called; resource.Update() returns error.
3. **Remove Space Member Not Implemented** - `removeSpaceMember` mutation exists but isn't integrated; deletion is a no-op.
4. **Roles and API Keys Data Sources Incomplete** - Not currently wired up; queries fail due to missing API endpoints.
5. **Acceptance Tests Not Implemented** - `resource_*_test.go` files have skeleton tests only.

## Development Workflow

### Build & Test

```bash
# Build the provider binary
make build

# Run unit tests
make test

# Run acceptance tests (requires ARIZE_API_KEY env var and TF_ACC=1)
make testacc

# Format code
make fmt

# Build and install to local Terraform plugin directory
make install
```

### Manual Testing with Terraform

```bash
# Set up dev override (so Terraform uses local binary instead of registry)
export TF_CLI_CONFIG_FILE=~/.terraform.d/credentials.tfrc

# Run terraform plan
cd examples/space_test
terraform plan

# Apply changes
terraform apply -auto-approve

# Inspect state
terraform state show arize_space.test
```

### Adding a New Resource

1. Create `internal/provider/resource_<name>.go` implementing:
   - `Metadata()` - sets type name
   - `Schema()` - defines HCL attributes and types
   - `Configure()` - initializes GraphQL client
   - `Create()`, `Read()`, `Update()`, `Delete()` - CRUD logic
   - `ImportState()` - optional, for `terraform import`

2. Add GraphQL operations to `internal/client/`:
   - Model struct in `models.go` or `models_space.go`
   - Query in `queries.go` (for Read)
   - Mutations in `mutations.go` (for Create/Update/Delete)

3. Register in `provider.go`:
   ```go
   func (p *ArizeProvider) Resources(ctx context.Context) []func() resource.Resource {
       return []func() resource.Resource{
           ...existing...
           NewMyResource,  // Add this
       }
   }
   ```

4. Test by creating a `.tf` file in `examples/` and running `terraform plan`.

## Environment & Configuration

**Required Environment Variables**
- `ARIZE_API_KEY` - API key for Arize account (can also be set in provider block)

**Terraform Dev Override** (for local testing)
```hcl
# ~/.terraform.d/credentials.tfrc
provider_installation {
  dev_overrides {
    "arize-ai/arize" = "/path/to/bin"
  }
  direct {}
}
```
Then run Terraform with: `TF_CLI_CONFIG_FILE=~/.terraform.d/credentials.tfrc terraform ...`

## Code Patterns & Conventions

**Resource Models**
- Terraform-side data: `*ResourceModel` (e.g., `UserResourceModel`)
- API-side data: Plain structs (e.g., `User`, `CreateUserInput`)
- Conversion happens in Create/Read/Update methods

**Error Handling**
- Client methods return `(result, error)`
- Resource methods call `resp.Diagnostics.AddError(title, message)` for user-facing errors
- GraphQL errors are wrapped with context: `fmt.Errorf("failed to create user: %w", err)`

**Null/Computed Values**
- Optional fields: `types.StringValue(value)` or `types.StringNull()`
- Computed-only fields: Set in Read/Create, use `PlanModifiers` for immutability
- Complex objects: Use `types.ObjectValue()` with attribute type map

## Debugging Tips

**Enable Terraform Debug Logging**
```bash
TF_LOG=DEBUG terraform plan
```

**Test GraphQL Queries Directly**
```bash
curl -X POST https://app.arize.com/graphql \
  -H "x-api-key: $ARIZE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query": "query { account { users(first: 1) { edges { node { id name email } } } } }"}'
```

**Check Introspection Schema**
Find available queries/mutations/types:
```bash
curl -X POST https://app.arize.com/graphql \
  -H "x-api-key: $ARIZE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query": "query { __type(name: \"Mutation\") { fields { name } } }"}'
```

## Common Tasks

**Running a Single Test**
```bash
# Not yet implemented, but would be:
go test -v -run TestAccUserResource ./internal/provider/
```

**Checking GraphQL Response Parsing**
The `client.execute()` method in `graphql.go` handles JSON unmarshaling. If a query fails to parse, add logging there.

**Validating Terraform Syntax**
```bash
terraform validate
```

## Related Files

- `README.md` - User-facing provider documentation
- `go.mod`, `go.sum` - Go dependencies (Terraform Plugin Framework v1.19.0+)
- `.terraformrc` - Dev override configuration (dev-only, don't commit)
- `examples/` - Reference Terraform configurations for each resource
