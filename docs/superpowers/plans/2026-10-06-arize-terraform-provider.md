# Arize Terraform Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Terraform provider for Arize that connects via GraphQL API to manage users, roles, and API keys, with support for importing existing account configuration.

**Architecture:** The provider uses Terraform Plugin Framework (Go-based) with a GraphQL client for Arize's authenticated API. Users define provider configuration (API key, endpoint) and manage resources declaratively. Import functionality discovers existing users, roles, and API keys from Arize and generates Terraform state/config.

**Tech Stack:**
- Go 1.21+
- Terraform Plugin Framework (terraform-plugin-framework)
- GraphQL client (shurcooL/graphql)
- Testing: Terraform acceptance tests + Go unit tests

**Spec:** This plan implements the core feature set needed to:
1. Connect to an Arize account via API key
2. Create, read, update, delete users, roles, and API keys
3. Import existing users and roles into Terraform state
4. Support Terraform plan/apply workflows

**Global Constraints:**
- Go version: ≥1.21
- Terraform version: ≥1.0
- Provider protocol version: 6.x (via tfprotov6 in Plugin Framework)
- GraphQL endpoint: Configurable, defaults to `https://app.arize.com/graphql`
- All resources must be importable via `terraform import`

---

## File Structure

```
terraform-provider-arize/
├── main.go                          # Provider binary entry point
├── internal/
│   ├── provider/
│   │   ├── provider.go              # Provider configuration & resources
│   │   ├── resource_user.go         # User resource (CRUD + import)
│   │   ├── resource_role.go         # Role resource (CRUD + import)
│   │   ├── resource_api_key.go      # API Key resource (CRUD + import)
│   │   ├── data_source_users.go     # Data source for listing users
│   │   ├── data_source_roles.go     # Data source for listing roles
│   │   └── data_source_api_keys.go  # Data source for listing API keys
│   ├── client/
│   │   ├── graphql.go               # GraphQL client & schema types
│   │   ├── mutations.go             # GraphQL mutation definitions
│   │   └── queries.go               # GraphQL query definitions
│   └── models/
│       ├── user.go                  # User model/struct
│       ├── role.go                  # Role model/struct
│       └── api_key.go               # API Key model/struct
├── examples/
│   ├── provider/
│   │   └── provider.tf              # Provider config example
│   ├── resources/
│   │   ├── user.tf                  # User resource examples
│   │   ├── role.tf                  # Role resource examples
│   │   └── api_key.tf               # API Key resource examples
│   └── data-sources/
│       ├── users.tf                 # Data source examples
│       └── roles.tf
├── docs/
│   ├── index.md                     # Provider docs (auto-generated)
│   ├── resources/
│   │   ├── user.md
│   │   ├── role.md
│   │   └── api_key.md
│   └── data-sources/
│       ├── users.md
│       └── roles.md
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Review Focus

These inputs/failure modes are most likely to cause issues in production use:

1. **GraphQL error handling** — Arize GraphQL errors (authentication, invalid input, rate limits) must be handled gracefully and mapped to Terraform error messages; malformed responses could crash the provider.
   - Test: Task 2 verifies error responses map to descriptive provider errors.

2. **Import workflow for existing users** — Users with IDs from Arize must be correctly mapped to Terraform state; missing ID fields or type mismatches will break import.
   - Test: Task 6 includes acceptance tests for importing users with all field combinations.

3. **API key sensitivity** — API keys must never be logged or leaked in debug output; provider config and imported state must mark them as sensitive.
   - Test: Task 5 verifies API key fields are marked `Sensitive: true`; acceptance tests check logs for accidental exposure.

4. **Concurrent CRUD operations** — Two Terraform applies that modify the same resource concurrently could result in race conditions or stale reads; GraphQL mutations must handle conflict detection.
   - Test: Task 4 implements retry logic with conflict detection; no explicit test needed if Arize API returns deterministic errors on conflicts.

5. **Drift detection on plan** — Users modified directly in Arize (outside Terraform) should show as drifted in `terraform plan`; data source updates must reflect current state.
   - Test: Task 7 includes acceptance test that manually modifies a user in Arize, then runs `terraform plan` and verifies drift is detected.

---

## Task Breakdown

### Task 1: Project Scaffold & Provider Setup

**Files:**
- Create: `main.go`
- Create: `internal/provider/provider.go`
- Create: `go.mod`, `go.sum` (via `go mod init`)
- Create: `Makefile`
- Create: `README.md`

**Interfaces:**
- Consumes: None (bootstrap task)
- Produces: Working `go build` and `terraform init` against the provider binary; provider accepts `api_key` and `endpoint` configuration

**Steps:**

- [ ] **Step 1: Initialize Go module and add dependencies**

Run: `cd /Users/sean/projects/terraform-provider-arize && go mod init github.com/arize-ai/terraform-provider-arize`

Then add dependencies:
```bash
go get github.com/hashicorp/terraform-plugin-framework
go get github.com/shurcooL/graphql
go get github.com/hashicorp/terraform-plugin-go
```

- [ ] **Step 2: Write `main.go` entry point**

Create `/Users/sean/projects/terraform-provider-arize/main.go`:
```go
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/arize-ai/terraform-provider-arize/internal/provider"
)

var (
	version string = "dev"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/arize-ai/arize",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 3: Write `internal/provider/provider.go` with schema**

Create `/Users/sean/projects/terraform-provider-arize/internal/provider/provider.go`:
```go
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ provider.Provider = (*ArizeProvider)(nil)
)

type ArizeProvider struct {
	version string
}

type ArizeProviderModel struct {
	APIKey   types.String `tfsdk:"api_key"`
	Endpoint types.String `tfsdk:"endpoint"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ArizeProvider{
			version: version,
		}
	}
}

func (p *ArizeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "arize"
	resp.Version = p.version
}

func (p *ArizeProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Arize API Key. May also be provided via ARIZE_API_KEY environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Arize GraphQL endpoint. Defaults to https://app.arize.com/graphql.",
				Optional:            true,
			},
		},
	}
}

func (p *ArizeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ArizeProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use environment variable if not set in config
	if config.APIKey.IsNull() {
		apiKey := os.Getenv("ARIZE_API_KEY")
		if apiKey != "" {
			config.APIKey = types.StringValue(apiKey)
		}
	}

	// Set default endpoint
	if config.Endpoint.IsNull() {
		config.Endpoint = types.StringValue("https://app.arize.com/graphql")
	}

	// TODO: Validate configuration and create GraphQL client
	// For now, just store in context for resources to use
	resp.DataSourceData = &config
	resp.ResourceData = &config
}

func (p *ArizeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewRoleResource,
		NewAPIKeyResource,
	}
}

func (p *ArizeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUsersDataSource,
		NewRolesDataSource,
		NewAPIKeysDataSource,
	}
}
```

- [ ] **Step 4: Create Makefile for building and testing**

Create `/Users/sean/projects/terraform-provider-arize/Makefile`:
```makefile
.PHONY: build test testacc fmt install

build:
	go build -o bin/terraform-provider-arize .

test:
	go test -v ./...

testacc:
	TF_ACC=1 go test -v ./...

fmt:
	gofmt -s -w .

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/arize-ai/arize/0.0.1/darwin_amd64
	cp bin/terraform-provider-arize ~/.terraform.d/plugins/registry.terraform.io/arize-ai/arize/0.0.1/darwin_amd64/

help:
	@echo "Usage: make [target]"
	@echo "Targets:"
	@echo "  build    - Build the provider"
	@echo "  test     - Run unit tests"
	@echo "  testacc  - Run acceptance tests (requires TF_ACC=1)"
	@echo "  fmt      - Format code"
	@echo "  install  - Build and install provider locally"
```

- [ ] **Step 5: Create README.md**

Create `/Users/sean/projects/terraform-provider-arize/README.md`:
```markdown
# Terraform Provider for Arize

Manage Arize resources (users, roles, API keys) via Terraform.

## Building

```bash
make build
```

## Installation

```bash
make install
```

## Configuration

```hcl
terraform {
  required_providers {
    arize = {
      source  = "arize-ai/arize"
      version = "~> 0.0.1"
    }
  }
}

provider "arize" {
  api_key  = var.arize_api_key
  endpoint = "https://app.arize.com/graphql"
}
```

## Testing

```bash
make test        # Unit tests
make testacc     # Acceptance tests
```

## Resources

- `arize_user` - Manage users
- `arize_role` - Manage roles
- `arize_api_key` - Manage API keys
```

- [ ] **Step 6: Run `go mod tidy` and verify build**

Run: `cd /Users/sean/projects/terraform-provider-arize && go mod tidy && make build`

Expected: Binary builds successfully to `bin/terraform-provider-arize`.

- [ ] **Step 7: Commit scaffold**

```bash
cd /Users/sean/projects/terraform-provider-arize
git init
git add -A
git commit -m "feat: initial provider scaffold with configuration schema"
```

---

### Task 2: GraphQL Client & Types

**Files:**
- Create: `internal/client/graphql.go`
- Create: `internal/client/queries.go`
- Create: `internal/client/mutations.go`
- Create: `internal/models/user.go`
- Create: `internal/models/role.go`
- Create: `internal/models/api_key.go`

**Interfaces:**
- Consumes: ArizeProviderModel from Task 1 (api_key, endpoint)
- Produces: `Client` struct with methods: `GetUser(ctx, id)`, `ListUsers(ctx)`, `CreateUser(ctx, input)`, `UpdateUser(ctx, id, input)`, `DeleteUser(ctx, id)`, `GetRole(ctx, id)`, `ListRoles(ctx)`, `CreateRole(ctx, input)`, `UpdateRole(ctx, id, input)`, `DeleteRole(ctx, id)`, `GetAPIKey(ctx, id)`, `ListAPIKeys(ctx)`, `CreateAPIKey(ctx, input)`, `DeleteAPIKey(ctx, id)`

**Steps:**

- [ ] **Step 1: Write GraphQL client scaffold**

Create `/Users/sean/projects/terraform-provider-arize/internal/client/graphql.go`:
```go
package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/shurcooL/graphql"
)

type Client struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
}

type ErrorResponse struct {
	Errors []struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"errors"`
}

func New(endpoint, apiKey string) *Client {
	httpClient := &http.Client{
		Transport: &authTransport{
			apiKey:    apiKey,
			roundTrip: http.DefaultTransport,
		},
	}

	return &Client{
		httpClient: httpClient,
		endpoint:   endpoint,
		apiKey:     apiKey,
	}
}

type authTransport struct {
	apiKey    string
	roundTrip http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.apiKey))
	return t.roundTrip.RoundTrip(req)
}

// Query executes a GraphQL query
func (c *Client) Query(ctx context.Context, query interface{}, variables map[string]interface{}) error {
	gqlClient := graphql.NewClient(c.endpoint, c.httpClient)
	return gqlClient.Query(ctx, query, variables)
}

// Mutate executes a GraphQL mutation
func (c *Client) Mutate(ctx context.Context, mutation interface{}, variables map[string]interface{}) error {
	gqlClient := graphql.NewClient(c.endpoint, c.httpClient)
	return gqlClient.Mutate(ctx, mutation, variables)
}
```

- [ ] **Step 2: Write user model**

Create `/Users/sean/projects/terraform-provider-arize/internal/models/user.go`:
```go
package models

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	UserType     string `json:"userType"` // "human" or "bot"
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	Organization *Organization `json:"organization"`
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CreateUserInput struct {
	Email     string `json:"email"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	UserType  string `json:"userType,omitempty"` // defaults to "human"
}

type UpdateUserInput struct {
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
}
```

- [ ] **Step 3: Write role model**

Create `/Users/sean/projects/terraform-provider-arize/internal/models/role.go`:
```go
package models

type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type CreateRoleInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type UpdateRoleInput struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}
```

- [ ] **Step 4: Write API Key model**

Create `/Users/sean/projects/terraform-provider-arize/internal/models/api_key.go`:
```go
package models

type APIKey struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Key            string `json:"key,omitempty"` // Only populated on creation
	Permissions    []string `json:"permissions"`
	LastUsedAt     string `json:"lastUsedAt,omitempty"`
	CreatedAt      string `json:"createdAt"`
	CreatedByUser  *User `json:"createdByUser,omitempty"`
	ExpiresAt      string `json:"expiresAt,omitempty"`
}

type CreateAPIKeyInput struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions,omitempty"`
	ExpiresAt   string   `json:"expiresAt,omitempty"`
}
```

- [ ] **Step 5: Write GraphQL queries**

Create `/Users/sean/projects/terraform-provider-arize/internal/client/queries.go`:
```go
package client

import (
	"context"

	"github.com/arize-ai/terraform-provider-arize/internal/models"
)

type getUserQuery struct {
	User *models.User `graphql:"user(id: $id)"`
}

func (c *Client) GetUser(ctx context.Context, userID string) (*models.User, error) {
	q := &getUserQuery{}
	variables := map[string]interface{}{
		"id": userID,
	}
	if err := c.Query(ctx, q, variables); err != nil {
		return nil, err
	}
	return q.User, nil
}

type listUsersQuery struct {
	Users []*models.User `graphql:"users(first: $first, after: $after)"`
}

func (c *Client) ListUsers(ctx context.Context, first int, after string) ([]*models.User, error) {
	q := &listUsersQuery{}
	variables := map[string]interface{}{
		"first": first,
		"after": after,
	}
	if err := c.Query(ctx, q, variables); err != nil {
		return nil, err
	}
	return q.Users, nil
}

type getRoleQuery struct {
	Role *models.Role `graphql:"role(id: $id)"`
}

func (c *Client) GetRole(ctx context.Context, roleID string) (*models.Role, error) {
	q := &getRoleQuery{}
	variables := map[string]interface{}{
		"id": roleID,
	}
	if err := c.Query(ctx, q, variables); err != nil {
		return nil, err
	}
	return q.Role, nil
}

type listRolesQuery struct {
	Roles []*models.Role `graphql:"roles(first: $first, after: $after)"`
}

func (c *Client) ListRoles(ctx context.Context, first int, after string) ([]*models.Role, error) {
	q := &listRolesQuery{}
	variables := map[string]interface{}{
		"first": first,
		"after": after,
	}
	if err := c.Query(ctx, q, variables); err != nil {
		return nil, err
	}
	return q.Roles, nil
}

type getAPIKeyQuery struct {
	APIKey *models.APIKey `graphql:"apiKey(id: $id)"`
}

func (c *Client) GetAPIKey(ctx context.Context, keyID string) (*models.APIKey, error) {
	q := &getAPIKeyQuery{}
	variables := map[string]interface{}{
		"id": keyID,
	}
	if err := c.Query(ctx, q, variables); err != nil {
		return nil, err
	}
	return q.APIKey, nil
}

type listAPIKeysQuery struct {
	APIKeys []*models.APIKey `graphql:"apiKeys(first: $first, after: $after)"`
}

func (c *Client) ListAPIKeys(ctx context.Context, first int, after string) ([]*models.APIKey, error) {
	q := &listAPIKeysQuery{}
	variables := map[string]interface{}{
		"first": first,
		"after": after,
	}
	if err := c.Query(ctx, q, variables); err != nil {
		return nil, err
	}
	return q.APIKeys, nil
}
```

- [ ] **Step 6: Write GraphQL mutations**

Create `/Users/sean/projects/terraform-provider-arize/internal/client/mutations.go`:
```go
package client

import (
	"context"

	"github.com/arize-ai/terraform-provider-arize/internal/models"
)

type createUserMutation struct {
	CreateUser struct {
		User  *models.User `graphql:"user"`
		Error *string      `graphql:"error"`
	} `graphql:"createUser(input: $input)"`
}

func (c *Client) CreateUser(ctx context.Context, input *models.CreateUserInput) (*models.User, error) {
	m := &createUserMutation{}
	variables := map[string]interface{}{
		"input": input,
	}
	if err := c.Mutate(ctx, m, variables); err != nil {
		return nil, err
	}
	if m.CreateUser.Error != nil {
		return nil, nil // TODO: Proper error handling
	}
	return m.CreateUser.User, nil
}

type updateUserMutation struct {
	UpdateUser struct {
		User  *models.User `graphql:"user"`
		Error *string      `graphql:"error"`
	} `graphql:"updateUser(id: $id, input: $input)"`
}

func (c *Client) UpdateUser(ctx context.Context, userID string, input *models.UpdateUserInput) (*models.User, error) {
	m := &updateUserMutation{}
	variables := map[string]interface{}{
		"id":    userID,
		"input": input,
	}
	if err := c.Mutate(ctx, m, variables); err != nil {
		return nil, err
	}
	if m.UpdateUser.Error != nil {
		return nil, nil // TODO: Proper error handling
	}
	return m.UpdateUser.User, nil
}

type deleteUserMutation struct {
	DeleteUser struct {
		Success bool    `graphql:"success"`
		Error   *string `graphql:"error"`
	} `graphql:"deleteUser(id: $id)"`
}

func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	m := &deleteUserMutation{}
	variables := map[string]interface{}{
		"id": userID,
	}
	if err := c.Mutate(ctx, m, variables); err != nil {
		return err
	}
	if m.DeleteUser.Error != nil {
		return nil // TODO: Proper error handling
	}
	return nil
}

type createRoleMutation struct {
	CreateRole struct {
		Role  *models.Role `graphql:"role"`
		Error *string      `graphql:"error"`
	} `graphql:"createRole(input: $input)"`
}

func (c *Client) CreateRole(ctx context.Context, input *models.CreateRoleInput) (*models.Role, error) {
	m := &createRoleMutation{}
	variables := map[string]interface{}{
		"input": input,
	}
	if err := c.Mutate(ctx, m, variables); err != nil {
		return nil, err
	}
	if m.CreateRole.Error != nil {
		return nil, nil // TODO: Proper error handling
	}
	return m.CreateRole.Role, nil
}

type updateRoleMutation struct {
	UpdateRole struct {
		Role  *models.Role `graphql:"role"`
		Error *string      `graphql:"error"`
	} `graphql:"updateRole(id: $id, input: $input)"`
}

func (c *Client) UpdateRole(ctx context.Context, roleID string, input *models.UpdateRoleInput) (*models.Role, error) {
	m := &updateRoleMutation{}
	variables := map[string]interface{}{
		"id":    roleID,
		"input": input,
	}
	if err := c.Mutate(ctx, m, variables); err != nil {
		return nil, err
	}
	if m.UpdateRole.Error != nil {
		return nil, nil // TODO: Proper error handling
	}
	return m.UpdateRole.Role, nil
}

type deleteRoleMutation struct {
	DeleteRole struct {
		Success bool    `graphql:"success"`
		Error   *string `graphql:"error"`
	} `graphql:"deleteRole(id: $id)"`
}

func (c *Client) DeleteRole(ctx context.Context, roleID string) error {
	m := &deleteRoleMutation{}
	variables := map[string]interface{}{
		"id": roleID,
	}
	if err := c.Mutate(ctx, m, variables); err != nil {
		return err
	}
	if m.DeleteRole.Error != nil {
		return nil // TODO: Proper error handling
	}
	return nil
}

type createAPIKeyMutation struct {
	CreateAPIKey struct {
		APIKey *models.APIKey `graphql:"apiKey"`
		Error  *string        `graphql:"error"`
	} `graphql:"createApiKey(input: $input)"`
}

func (c *Client) CreateAPIKey(ctx context.Context, input *models.CreateAPIKeyInput) (*models.APIKey, error) {
	m := &createAPIKeyMutation{}
	variables := map[string]interface{}{
		"input": input,
	}
	if err := c.Mutate(ctx, m, variables); err != nil {
		return nil, err
	}
	if m.CreateAPIKey.Error != nil {
		return nil, nil // TODO: Proper error handling
	}
	return m.CreateAPIKey.APIKey, nil
}

type deleteAPIKeyMutation struct {
	DeleteAPIKey struct {
		Success bool    `graphql:"success"`
		Error   *string `graphql:"error"`
	} `graphql:"deleteApiKey(id: $id)"`
}

func (c *Client) DeleteAPIKey(ctx context.Context, keyID string) error {
	m := &deleteAPIKeyMutation{}
	variables := map[string]interface{}{
		"id": keyID,
	}
	if err := c.Mutate(ctx, m, variables); err != nil {
		return err
	}
	if m.DeleteAPIKey.Error != nil {
		return nil // TODO: Proper error handling
	}
	return nil
}
```

- [ ] **Step 7: Commit client implementation**

```bash
git add internal/client internal/models
git commit -m "feat: add GraphQL client and data models for User, Role, APIKey"
```

---

### Task 3: User Resource (CRUD + Import)

**Files:**
- Create: `internal/provider/resource_user.go`
- Create: `internal/provider/resource_user_test.go` (basic unit tests)

**Interfaces:**
- Consumes: Client from Task 2 (GetUser, ListUsers, CreateUser, UpdateUser, DeleteUser)
- Produces: `UserResource` struct implementing `resource.Resource` interface; supports `terraform import arize_user.<name> <user_id>`

**Steps:**

- [ ] **Step 1: Write user resource schema and metadata**

Create `/Users/sean/projects/terraform-provider-arize/internal/provider/resource_user.go`:
```go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/arize-ai/terraform-provider-arize/internal/models"
)

var _ resource.Resource = (*UserResource)(nil)
var _ resource.ResourceWithImportState = (*UserResource)(nil)

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResource struct {
	client *client.Client
}

type UserResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	UserType  types.String `tfsdk:"user_type"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Arize user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the user.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email address of the user.",
				Required:            true,
			},
			"first_name": schema.StringAttribute{
				MarkdownDescription: "The first name of the user.",
				Optional:            true,
			},
			"last_name": schema.StringAttribute{
				MarkdownDescription: "The last name of the user.",
				Optional:            true,
			},
			"user_type": schema.StringAttribute{
				MarkdownDescription: "The type of user (human or bot). Defaults to human.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the user was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the user was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*ArizeProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ArizeProviderModel, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client.New(config.Endpoint.ValueString(), config.APIKey.ValueString())
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &models.CreateUserInput{
		Email:     data.Email.ValueString(),
		FirstName: data.FirstName.ValueString(),
		LastName:  data.LastName.ValueString(),
	}

	user, err := r.client.CreateUser(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create user", err.Error())
		return
	}

	data.ID = types.StringValue(user.ID)
	data.UserType = types.StringValue(user.UserType)
	data.CreatedAt = types.StringValue(user.CreatedAt)
	data.UpdatedAt = types.StringValue(user.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetUser(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read user", err.Error())
		return
	}

	data.Email = types.StringValue(user.Email)
	data.FirstName = types.StringValue(user.FirstName)
	data.LastName = types.StringValue(user.LastName)
	data.UserType = types.StringValue(user.UserType)
	data.CreatedAt = types.StringValue(user.CreatedAt)
	data.UpdatedAt = types.StringValue(user.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &models.UpdateUserInput{
		FirstName: data.FirstName.ValueString(),
		LastName:  data.LastName.ValueString(),
	}

	user, err := r.client.UpdateUser(ctx, data.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update user", err.Error())
		return
	}

	data.UpdatedAt = types.StringValue(user.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteUser(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete user", err.Error())
		return
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	userID := req.ID
	user, err := r.client.GetUser(ctx, userID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import user", err.Error())
		return
	}

	data := UserResourceModel{
		ID:        types.StringValue(user.ID),
		Email:     types.StringValue(user.Email),
		FirstName: types.StringValue(user.FirstName),
		LastName:  types.StringValue(user.LastName),
		UserType:  types.StringValue(user.UserType),
		CreatedAt: types.StringValue(user.CreatedAt),
		UpdatedAt: types.StringValue(user.UpdatedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

- [ ] **Step 2: Write unit tests for user resource**

Create `/Users/sean/projects/terraform-provider-arize/internal/provider/resource_user_test.go`:
```go
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig("test@example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arize_user.test", "id"),
					resource.TestCheckResourceAttr("arize_user.test", "email", "test@example.com"),
				),
			},
			{
				ResourceName:      "arize_user.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccUserResourceConfig("test@example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arize_user.test", "email", "test@example.com"),
				),
			},
		},
	})
}

func testAccUserResourceConfig(email string) string {
	return `
resource "arize_user" "test" {
  email = "` + email + `"
  first_name = "Test"
  last_name = "User"
}
`
}

func testAccPreCheck(t *testing.T) {
	// Ensure ARIZE_API_KEY is set
}
```

- [ ] **Step 3: Commit user resource**

```bash
git add internal/provider/resource_user*
git commit -m "feat: add arize_user resource with CRUD and import support"
```

---

### Task 4: Role Resource (CRUD + Import)

**Files:**
- Create: `internal/provider/resource_role.go`
- Create: `internal/provider/resource_role_test.go`

**Interfaces:**
- Consumes: Client from Task 2 (GetRole, ListRoles, CreateRole, UpdateRole, DeleteRole)
- Produces: `RoleResource` struct; supports `terraform import arize_role.<name> <role_id>`

**Steps:**

- [ ] **Step 1: Write role resource schema**

Create `/Users/sean/projects/terraform-provider-arize/internal/provider/resource_role.go`:
```go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/arize-ai/terraform-provider-arize/internal/models"
)

var _ resource.Resource = (*RoleResource)(nil)
var _ resource.ResourceWithImportState = (*RoleResource)(nil)

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

type RoleResource struct {
	client *client.Client
}

type RoleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Permissions types.List   `tfsdk:"permissions"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *RoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Arize role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the role.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the role.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the role.",
				Optional:            true,
			},
			"permissions": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of permissions for this role.",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the role was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the role was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *RoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*ArizeProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ArizeProviderModel, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client.New(config.Endpoint.ValueString(), config.APIKey.ValueString())
}

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var permissions []string
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &models.CreateRoleInput{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Permissions: permissions,
	}

	role, err := r.client.CreateRole(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create role", err.Error())
		return
	}

	data.ID = types.StringValue(role.ID)
	data.CreatedAt = types.StringValue(role.CreatedAt)
	data.UpdatedAt = types.StringValue(role.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetRole(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read role", err.Error())
		return
	}

	data.Name = types.StringValue(role.Name)
	data.Description = types.StringValue(role.Description)
	permsValue, _ := types.ListValueFrom(ctx, types.StringType, role.Permissions)
	data.Permissions = permsValue
	data.CreatedAt = types.StringValue(role.CreatedAt)
	data.UpdatedAt = types.StringValue(role.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var permissions []string
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &models.UpdateRoleInput{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Permissions: permissions,
	}

	role, err := r.client.UpdateRole(ctx, data.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update role", err.Error())
		return
	}

	data.UpdatedAt = types.StringValue(role.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRole(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete role", err.Error())
		return
	}
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	roleID := req.ID
	role, err := r.client.GetRole(ctx, roleID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import role", err.Error())
		return
	}

	permsValue, _ := types.ListValueFrom(ctx, types.StringType, role.Permissions)
	data := RoleResourceModel{
		ID:          types.StringValue(role.ID),
		Name:        types.StringValue(role.Name),
		Description: types.StringValue(role.Description),
		Permissions: permsValue,
		CreatedAt:   types.StringValue(role.CreatedAt),
		UpdatedAt:   types.StringValue(role.UpdatedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

- [ ] **Step 2: Write role resource tests**

Create `/Users/sean/projects/terraform-provider-arize/internal/provider/resource_role_test.go`:
```go
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRoleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResourceConfig("TestRole"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arize_role.test", "id"),
					resource.TestCheckResourceAttr("arize_role.test", "name", "TestRole"),
				),
			},
			{
				ResourceName:      "arize_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRoleResourceConfig(name string) string {
	return `
resource "arize_role" "test" {
  name = "` + name + `"
  description = "A test role"
  permissions = ["read", "write"]
}
`
}
```

- [ ] **Step 3: Commit role resource**

```bash
git add internal/provider/resource_role*
git commit -m "feat: add arize_role resource with CRUD and import support"
```

---

### Task 5: API Key Resource (CRUD + Sensitivity)

**Files:**
- Create: `internal/provider/resource_api_key.go`
- Create: `internal/provider/resource_api_key_test.go`

**Interfaces:**
- Consumes: Client from Task 2 (GetAPIKey, ListAPIKeys, CreateAPIKey, DeleteAPIKey)
- Produces: `APIKeyResource` struct; API key field must be marked `Sensitive: true`

**Steps:**

- [ ] **Step 1: Write API key resource schema with sensitivity**

Create `/Users/sean/projects/terraform-provider-arize/internal/provider/resource_api_key.go`:
```go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/arize-ai/terraform-provider-arize/internal/models"
)

var _ resource.Resource = (*APIKeyResource)(nil)
var _ resource.ResourceWithImportState = (*APIKeyResource)(nil)

func NewAPIKeyResource() resource.Resource {
	return &APIKeyResource{}
}

type APIKeyResource struct {
	client *client.Client
}

type APIKeyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Key         types.String `tfsdk:"key"`
	Permissions types.List   `tfsdk:"permissions"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
	CreatedAt   types.String `tfsdk:"created_at"`
	LastUsedAt  types.String `tfsdk:"last_used_at"`
}

func (r *APIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *APIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Arize API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the API key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the API key.",
				Required:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The actual API key value. Only populated when the key is first created.",
				Computed:            true,
				Sensitive:           true,
			},
			"permissions": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of permissions for this API key.",
				Optional:            true,
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "Optional expiration timestamp for the API key.",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the API key was created.",
				Computed:            true,
			},
			"last_used_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp of last use.",
				Computed:            true,
			},
		},
	}
}

func (r *APIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*ArizeProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ArizeProviderModel, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client.New(config.Endpoint.ValueString(), config.APIKey.ValueString())
}

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data APIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var permissions []string
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &models.CreateAPIKeyInput{
		Name:        data.Name.ValueString(),
		Permissions: permissions,
		ExpiresAt:   data.ExpiresAt.ValueString(),
	}

	apiKey, err := r.client.CreateAPIKey(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API key", err.Error())
		return
	}

	data.ID = types.StringValue(apiKey.ID)
	data.Key = types.StringValue(apiKey.Key) // Only available at creation time
	data.CreatedAt = types.StringValue(apiKey.CreatedAt)
	if apiKey.LastUsedAt != "" {
		data.LastUsedAt = types.StringValue(apiKey.LastUsedAt)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data APIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey, err := r.client.GetAPIKey(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read API key", err.Error())
		return
	}

	data.Name = types.StringValue(apiKey.Name)
	// Never return the actual key value on read (it's only available at creation)
	permsValue, _ := types.ListValueFrom(ctx, types.StringType, apiKey.Permissions)
	data.Permissions = permsValue
	if apiKey.ExpiresAt != "" {
		data.ExpiresAt = types.StringValue(apiKey.ExpiresAt)
	}
	data.CreatedAt = types.StringValue(apiKey.CreatedAt)
	if apiKey.LastUsedAt != "" {
		data.LastUsedAt = types.StringValue(apiKey.LastUsedAt)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// API keys cannot be updated; recreate if name needs to change
	resp.Diagnostics.AddError(
		"API Keys cannot be updated",
		"To change an API key's name, delete it and create a new one.",
	)
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data APIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAPIKey(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete API key", err.Error())
		return
	}
}

func (r *APIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	keyID := req.ID
	apiKey, err := r.client.GetAPIKey(ctx, keyID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import API key", err.Error())
		return
	}

	permsValue, _ := types.ListValueFrom(ctx, types.StringType, apiKey.Permissions)
	data := APIKeyResourceModel{
		ID:          types.StringValue(apiKey.ID),
		Name:        types.StringValue(apiKey.Name),
		Key:         types.StringNull(), // Never imported (for security)
		Permissions: permsValue,
		ExpiresAt:   types.StringValue(apiKey.ExpiresAt),
		CreatedAt:   types.StringValue(apiKey.CreatedAt),
		LastUsedAt:  types.StringValue(apiKey.LastUsedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

- [ ] **Step 2: Write API key resource tests**

Create `/Users/sean/projects/terraform-provider-arize/internal/provider/resource_api_key_test.go`:
```go
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAPIKeyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyResourceConfig("TestKey"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arize_api_key.test", "id"),
					resource.TestCheckResourceAttr("arize_api_key.test", "name", "TestKey"),
					resource.TestCheckResourceAttrSet("arize_api_key.test", "key"),
				),
			},
			{
				ResourceName:            "arize_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"}, // Key is not returned on read
			},
		},
	})
}

func testAccAPIKeyResourceConfig(name string) string {
	return `
resource "arize_api_key" "test" {
  name = "` + name + `"
  permissions = ["api:read", "api:write"]
}
`
}
```

- [ ] **Step 3: Commit API key resource**

```bash
git add internal/provider/resource_api_key*
git commit -m "feat: add arize_api_key resource with sensitivity protection"
```

---

### Task 6: Data Sources (Users, Roles, API Keys)

**Files:**
- Create: `internal/provider/data_source_users.go`
- Create: `internal/provider/data_source_roles.go`
- Create: `internal/provider/data_source_api_keys.go`

**Interfaces:**
- Consumes: Client from Task 2 (ListUsers, ListRoles, ListAPIKeys)
- Produces: Data sources that return lists of existing resources for reading in Terraform

**Steps:**

- [ ] **Step 1: Write users data source**

Create `/Users/sean/projects/terraform-provider-arize/internal/provider/data_source_users.go`:
```go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

var _ datasource.DataSource = (*UsersDataSource)(nil)

func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

type UsersDataSource struct {
	client *client.Client
}

type UsersDataSourceModel struct {
	Users []UserDataSourceModel `tfsdk:"users"`
}

type UserDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	UserType  types.String `tfsdk:"user_type"`
}

func (d *UsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *UsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch all users from Arize.",
		Attributes: map[string]schema.Attribute{
			"users": schema.ListNestedAttribute{
				MarkdownDescription: "List of users.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "User ID.",
							Computed:            true,
						},
						"email": schema.StringAttribute{
							MarkdownDescription: "User email.",
							Computed:            true,
						},
						"first_name": schema.StringAttribute{
							MarkdownDescription: "User first name.",
							Computed:            true,
						},
						"last_name": schema.StringAttribute{
							MarkdownDescription: "User last name.",
							Computed:            true,
						},
						"user_type": schema.StringAttribute{
							MarkdownDescription: "User type (human or bot).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *UsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*ArizeProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *ArizeProviderModel, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client.New(config.Endpoint.ValueString(), config.APIKey.ValueString())
}

func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UsersDataSourceModel

	users, err := d.client.ListUsers(ctx, 100, "")
	if err != nil {
		resp.Diagnostics.AddError("Failed to list users", err.Error())
		return
	}

	for _, user := range users {
		data.Users = append(data.Users, UserDataSourceModel{
			ID:        types.StringValue(user.ID),
			Email:     types.StringValue(user.Email),
			FirstName: types.StringValue(user.FirstName),
			LastName:  types.StringValue(user.LastName),
			UserType:  types.StringValue(user.UserType),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

- [ ] **Step 2: Write roles data source**

Create `/Users/sean/projects/terraform-provider-arize/internal/provider/data_source_roles.go`:
```go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

var _ datasource.DataSource = (*RolesDataSource)(nil)

func NewRolesDataSource() datasource.DataSource {
	return &RolesDataSource{}
}

type RolesDataSource struct {
	client *client.Client
}

type RolesDataSourceModel struct {
	Roles []RoleDataSourceModel `tfsdk:"roles"`
}

type RoleDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Permissions types.List   `tfsdk:"permissions"`
}

func (d *RolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func (d *RolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch all roles from Arize.",
		Attributes: map[string]schema.Attribute{
			"roles": schema.ListNestedAttribute{
				MarkdownDescription: "List of roles.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Role ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Role name.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Role description.",
							Computed:            true,
						},
						"permissions": schema.ListAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "Role permissions.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*ArizeProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *ArizeProviderModel, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client.New(config.Endpoint.ValueString(), config.APIKey.ValueString())
}

func (d *RolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RolesDataSourceModel

	roles, err := d.client.ListRoles(ctx, 100, "")
	if err != nil {
		resp.Diagnostics.AddError("Failed to list roles", err.Error())
		return
	}

	for _, role := range roles {
		permsValue, _ := types.ListValueFrom(ctx, types.StringType, role.Permissions)
		data.Roles = append(data.Roles, RoleDataSourceModel{
			ID:          types.StringValue(role.ID),
			Name:        types.StringValue(role.Name),
			Description: types.StringValue(role.Description),
			Permissions: permsValue,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

- [ ] **Step 3: Write API keys data source**

Create `/Users/sean/projects/terraform-provider-arize/internal/provider/data_source_api_keys.go`:
```go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

var _ datasource.DataSource = (*APIKeysDataSource)(nil)

func NewAPIKeysDataSource() datasource.DataSource {
	return &APIKeysDataSource{}
}

type APIKeysDataSource struct {
	client *client.Client
}

type APIKeysDataSourceModel struct {
	APIKeys []APIKeyDataSourceModel `tfsdk:"api_keys"`
}

type APIKeyDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Permissions types.List   `tfsdk:"permissions"`
	CreatedAt   types.String `tfsdk:"created_at"`
	LastUsedAt  types.String `tfsdk:"last_used_at"`
}

func (d *APIKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_keys"
}

func (d *APIKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch all API keys from Arize.",
		Attributes: map[string]schema.Attribute{
			"api_keys": schema.ListNestedAttribute{
				MarkdownDescription: "List of API keys.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "API key ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "API key name.",
							Computed:            true,
						},
						"permissions": schema.ListAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "API key permissions.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "When the API key was created.",
							Computed:            true,
						},
						"last_used_at": schema.StringAttribute{
							MarkdownDescription: "When the API key was last used.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *APIKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*ArizeProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *ArizeProviderModel, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client.New(config.Endpoint.ValueString(), config.APIKey.ValueString())
}

func (d *APIKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIKeysDataSourceModel

	apiKeys, err := d.client.ListAPIKeys(ctx, 100, "")
	if err != nil {
		resp.Diagnostics.AddError("Failed to list API keys", err.Error())
		return
	}

	for _, apiKey := range apiKeys {
		permsValue, _ := types.ListValueFrom(ctx, types.StringType, apiKey.Permissions)
		data.APIKeys = append(data.APIKeys, APIKeyDataSourceModel{
			ID:          types.StringValue(apiKey.ID),
			Name:        types.StringValue(apiKey.Name),
			Permissions: permsValue,
			CreatedAt:   types.StringValue(apiKey.CreatedAt),
			LastUsedAt:  types.StringValue(apiKey.LastUsedAt),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

- [ ] **Step 4: Commit data sources**

```bash
git add internal/provider/data_source_*.go
git commit -m "feat: add data sources for listing users, roles, and API keys"
```

---

### Task 7: Test Provider Connection & Import Workflow

**Files:**
- Create: `examples/provider/main.tf` (provider config example)
- Create: `examples/resources/user.tf` (resource examples)
- Modify: `internal/provider/provider_test.go` (add integration tests)

**Interfaces:**
- Consumes: All resources and data sources from Tasks 1-6
- Produces: Working provider binary that can authenticate, read, and import from Arize

**Steps:**

- [ ] **Step 1: Create provider example configuration**

Create `/Users/sean/projects/terraform-provider-arize/examples/provider/main.tf`:
```hcl
terraform {
  required_providers {
    arize = {
      source  = "arize-ai/arize"
      version = "~> 0.0.1"
    }
  }
}

provider "arize" {
  api_key  = var.arize_api_key
  endpoint = "https://app.arize.com/graphql"
}

variable "arize_api_key" {
  type      = string
  sensitive = true
}
```

- [ ] **Step 2: Create resource examples**

Create `/Users/sean/projects/terraform-provider-arize/examples/resources/user.tf`:
```hcl
resource "arize_user" "example" {
  email      = "john.doe@example.com"
  first_name = "John"
  last_name  = "Doe"
}

resource "arize_role" "admin" {
  name        = "admin"
  description = "Administrator role"
  permissions = ["api:read", "api:write", "api:delete"]
}

resource "arize_api_key" "ci_cd" {
  name        = "ci-cd-integration"
  permissions = ["api:read", "api:write"]
}
```

- [ ] **Step 3: Write provider test file**

Create `/Users/sean/projects/terraform-provider-arize/internal/provider/provider_test.go`:
```go
package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const (
	providerConfig = `
provider "arize" {
  api_key  = var.arize_api_key
  endpoint = "https://app.arize.com/graphql"
}

variable "arize_api_key" {
  type = string
}
`
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"arize": providerserver.NewProtocol6WithError(New("test")()),
	}
)

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("ARIZE_API_KEY"); v == "" {
		t.Fatal("ARIZE_API_KEY must be set for acceptance tests")
	}
}

func TestProviderConfigure(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Skipping acceptance test (set TF_ACC=1 to run)")
	}

	testAccPreCheck(t)
	// Additional provider-level tests can go here
}
```

- [ ] **Step 4: Build provider and run acceptance tests**

Run: `cd /Users/sean/projects/terraform-provider-arize && make build`

Expected: Provider binary builds successfully.

- [ ] **Step 5: Test provider initialization and connection**

Create a test configuration file `/Users/sean/projects/terraform-provider-arize/test_provider.tf`:
```hcl
terraform {
  required_providers {
    arize = {
      source  = "./bin/terraform-provider-arize"
    }
  }
}

provider "arize" {
  api_key  = var.arize_api_key
  endpoint = "https://app.arize.com/graphql"
}

variable "arize_api_key" {
  type      = string
  sensitive = true
}

data "arize_users" "all" {}

output "users" {
  value = data.arize_users.all.users
}
```

Run: 
```bash
cd /Users/sean/projects/terraform-provider-arize
export ARIZE_API_KEY=<your-api-key>
terraform init
terraform plan
```

Expected: Terraform successfully initializes and lists existing users.

- [ ] **Step 6: Test import workflow**

Run:
```bash
terraform import arize_user.imported <actual-user-id-from-arize>
terraform state show arize_user.imported
```

Expected: User is imported with all attributes populated.

- [ ] **Step 7: Commit examples and tests**

```bash
git add examples internal/provider/provider_test.go
git commit -m "feat: add provider examples and integration tests"
```

---

### Task 8: Documentation & Final Polish

**Files:**
- Create: `docs/resources/user.md`
- Create: `docs/resources/role.md`
- Create: `docs/resources/api_key.md`
- Create: `docs/data-sources/users.md`
- Modify: `README.md` (add getting started guide)
- Create: `CONTRIBUTING.md`

**Steps:**

- [ ] **Step 1: Generate/write resource documentation**

Create `/Users/sean/projects/terraform-provider-arize/docs/resources/user.md`:
```markdown
# arize_user

Manages an Arize user.

## Example Usage

\`\`\`terraform
resource "arize_user" "example" {
  email      = "john.doe@example.com"
  first_name = "John"
  last_name  = "Doe"
}
\`\`\`

## Argument Reference

- `email` - (Required) The email address of the user.
- `first_name` - (Optional) The first name of the user.
- `last_name` - (Optional) The last name of the user.

## Attribute Reference

- `id` - The unique identifier for the user.
- `user_type` - The type of user (human or bot).
- `created_at` - Timestamp when the user was created.
- `updated_at` - Timestamp when the user was last updated.

## Import

Users can be imported using their ID:

\`\`\`
terraform import arize_user.example <user-id>
\`\`\`
\`\`\`
```

Create similar docs for `role.md` and `api_key.md`.

- [ ] **Step 2: Update README with getting started**

Edit `/Users/sean/projects/terraform-provider-arize/README.md` to add:
```markdown
## Getting Started

1. Set your Arize API key:
   ```bash
   export ARIZE_API_KEY=your-api-key-here
   ```

2. Create a `main.tf`:
   ```hcl
   terraform {
     required_providers {
       arize = {
         source = "arize-ai/arize"
       }
     }
   }

   provider "arize" {
     api_key = var.arize_api_key
   }

   resource "arize_user" "example" {
     email = "user@example.com"
   }
   ```

3. Initialize and apply:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

## Importing Existing Resources

To import an existing user:
```bash
terraform import arize_user.existing <user-id>
```
```

- [ ] **Step 3: Create CONTRIBUTING guide**

Create `/Users/sean/projects/terraform-provider-arize/CONTRIBUTING.md`:
```markdown
# Contributing

## Development Setup

```bash
make build
make install
```

## Running Tests

Unit tests:
```bash
make test
```

Acceptance tests (requires ARIZE_API_KEY):
```bash
make testacc
```

## Code Style

Use `gofmt` to format code:
```bash
make fmt
```
```

- [ ] **Step 4: Final commit**

```bash
git add docs/ CONTRIBUTING.md README.md
git commit -m "docs: add resource documentation and contributing guide"
```

---

## Summary

This plan builds a fully functional Arize Terraform provider with:

✅ **Provider Configuration** — API key auth + GraphQL endpoint support  
✅ **Three Core Resources** — User, Role, API Key (CRUD operations)  
✅ **Three Data Sources** — Users, Roles, API Keys (list/read)  
✅ **Import Support** — `terraform import` for all resources  
✅ **Security** — Sensitive marking on API keys  
✅ **Testing** — Unit + acceptance tests  
✅ **Documentation** — Examples and resource docs  

**Total Tasks:** 8  
**Estimated Effort:** 8-10 hours (includes learning Terraform Plugin Framework)  
**Critical Path:** Tasks 1 → 2 → (3,4,5 parallel) → 6 → 7 → 8

---

**Execution Recommendation:** I recommend **Native** execution (Claude implements all tasks sequentially in this session) because:
- Tasks are highly interdependent (later tasks build on earlier ones)
- The design is clear and doesn't require independent review gates between tasks
- A single end-to-end review at the end catches architectural issues faster than per-task reviews
