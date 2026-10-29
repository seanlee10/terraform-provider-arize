# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Terraform provider for Arize, enabling Infrastructure as Code management of Arize resources (users, custom roles, API keys, spaces, space memberships, SAML IdPs, and evaluators). It uses the Terraform Plugin Framework (Go-based) and communicates with Arize's GraphQL API, plus the REST API for custom roles.

## Architecture

### High-Level Design

```
User Terraform Config (.tf files)
         ↓
Terraform Plugin Framework
         ↓
Provider (provider.go)
  ├── Resources (7 total)
  ├── Data Sources (4 total)
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
- `models.go` - Data structures for users, API keys
- `rest.go`, `roles.go`, `spaces.go` - REST client (Bearer auth), custom role CRUD, and space deletion (the GraphQL `deleteSpace` mutation returns an internal server error)
- `models_space.go` - Data structures for spaces and space members

### Critical API Knowledge

**Authentication Header**
The GraphQL API uses the `x-api-key` header, NOT `Authorization: Bearer`. This is different from many GraphQL APIs and is easy to get wrong. The REST API (`api.arize.com/v2`, `client/rest.go`) uses `Authorization: Bearer <same key>`.

A rejected or empty key gets HTTP 401 with a plain-text `Unauthorized` body (not JSON), so `execute()` checks the status before parsing. A key with a trailing newline gets 400.

**Regions**
Keys only work on their own cluster; a valid key sent to another cluster gets 401. EU users must set both `endpoint = "https://app.eu-west-1a.arize.com/graphql"` and `rest_endpoint = "https://api.eu-west-1a.arize.com"`.

**Query Patterns**
- **List resources**: Via `account.users`, `space.evaluators`, etc. (not root-level queries)
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

### Spaces (`resource_space.go`)

- An account can have several organizations; `organization_id` picks one (default: the first from `account.organizations`) and forces replacement on change.
- An unset description reads back as `""`; Read keeps it null, otherwise every plan shows an update and Update is unsupported.

### Space Roles

Valid space roles: `admin`, `member`, `readOnly`, `annotator`. These are case-sensitive enum values used in mutations.

### Roles (`resource_role.go`, `client/roles.go`)

GraphQL has no role list or mutations (roles are reachable only via `node(id)` and `SpaceUser.customRole`), so roles use the REST API `/v2/roles`:
- `PATCH` replaces `permissions` as a whole and `"description": null` clears the description; `UpdateRoleInput` always sends all fields.
- Errors are problem-details JSON (`{status, title, detail}`): 409 on duplicate name, 422 on unknown permission, 403 on predefined roles, 404 when missing.
- Predefined roles (`is_predefined`) are read-only; Read errors if a managed role turns out to be predefined.
- `permissions` is a Set since the server's order isn't meaningful.

### SAML IdP (`resource_saml_idp.go`, `client/mutations_saml.go`)

Verified against the live API:
- No delete mutation exists; Delete warns and removes from state only.
- `node(id)` returns null for SAML IdPs, so `GetSAMLIdP` searches `account.samlIdPs`.
- `updateSAMLIdP` replaces the whole IdP: omitted fields reset (email domains, role mappings, defaults). Every apply sends the full config, so the optional attributes are plain Optional (removing one clears it).
- Payloads are `{ error, idp }`; validation failures arrive in `error`, not GraphQL errors. "An error occurred while configuring SAML authentication" usually means an invalid role ID.
- `default_org_role_id` / `default_space_role_id` and mapping `orgRole.roleId` accept only built-in names (`admin`, `member`, `readOnly`, `annotator`); custom role IDs work only in `spaceRbacRolesMap` / `projectRolesMap`.
- `allowLoginWithDefaults: true` requires all four `default*` fields; `false` requires at least one role mapping, and each mapping needs `orgRole` unless `isAccountAdmin`.
- `*Map` fields are `[key, value]` tuple lists; the server sorts `attributesMap` by key. Unset strings come back as `""`.

### Evaluators (`resource_evaluator.go`, `client/evaluators.go`)

API behavior verified against the live API, which the resource depends on:
- Versioned: `editEvaluator` changes only name/description; any config change needs `createEvaluatorVersion` (commit message required). Update compares serialized config inputs to decide whether to publish a version.
- `dataGranularityType` cannot change after creation, so it is `RequiresReplace`.
- Evaluator names and result column names (`config.name`) are unique per space.
- `node(id)` on a deleted evaluator returns "Unauthorized access", not null; `GetEvaluator` confirms absence via `Space.evaluators` before returning `ErrEvaluatorNotFound`.
- `Space.evaluators(first: 100)` exceeds the query complexity limit (1000); page size is 50.
- `llmConfig` input requires all four fields (`integrationId`, `modelName`, `invocationParameters`, `providerParameters`); the server rewrites invocation parameters per provider, so Read keeps the configured values.
- `rails` is ignored by the provider: string input is stored as a single literal element; labels come from `classificationChoices`.
- Code `spanAttributes` is derived from the `evaluate()` signature (input is ignored), so it is computed.
- `editEvaluator` cannot clear a description (`""` and `null` are no-ops), so `description` is Optional+Computed with a ModifyPlan warning.
- Mutation payloads return `config: null`; read back with `GetEvaluator` after create/update.

### Known Limitations

1. **Space Updates Not Implemented** - `updateSpace` mutation exists but isn't called; resource.Update() returns error.
2. **Remove Space Member Not Implemented** - `removeSpaceMember` mutation exists but isn't integrated; deletion is a no-op.
3. **API Keys Data Source Incomplete** - Not currently wired up; queries fail due to missing API endpoints.
4. **Acceptance Tests Not Implemented** - `resource_*_test.go` files have skeleton tests only.
5. **Evaluator Types** - Only template and code evaluators are supported; harness, remote, and System One evaluators are not.

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
# (as arize-ai/arize at the latest git tag, for Terraform's platform, which can differ from Go's)
make install
```

### Distribution & Releases

The provider is not on the Terraform Registry, so `terraform init` cannot download it. Users install it locally with `make install`, a release zip placed in `~/.terraform.d/plugins/registry.terraform.io/arize-ai/arize/`, or a dev override. Releases are GitHub releases built locally with goreleaser:

```bash
git tag vX.Y.Z && git push origin main vX.Y.Z
GITHUB_TOKEN=$(gh auth token) GPG_FINGERPRINT=<signing key> goreleaser release --clean
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
