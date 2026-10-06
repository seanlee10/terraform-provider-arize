# Terraform Provider for Arize

[![GitHub release](https://img.shields.io/github/v/release/seanlee10/terraform-provider-arize?style=flat-square)](https://github.com/seanlee10/terraform-provider-arize/releases)

Manage your Arize account infrastructure as code with Terraform. Create and manage users, spaces, API keys, SAML/SSO configuration, and team access with full GitOps support.

## Features

✅ **User Management** - Create users, query existing users, manage access  
✅ **Space Management** - Create and manage ML model monitoring spaces  
✅ **Team Access Control** - Assign users to spaces with role-based permissions  
✅ **SAML/SSO** - Configure SAML identity providers and role mappings for enterprise SSO  
✅ **API Keys** - Generate and manage service account keys for automation  
✅ **Multi-Environment** - Deploy to dev, staging, and production with safety checks  
✅ **Import Existing** - Adopt Terraform for existing Arize infrastructure  
✅ **Data Sources** - Query and reference existing resources in your config  

## Quick Start

### 1. Get Your API Key

Generate an API key in your Arize account settings.

### 2. Configure Terraform

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
  # API key from ARIZE_API_KEY environment variable
  # or set explicitly:
  # api_key = var.arize_api_key
}

# Create a user
resource "arize_user" "alice" {
  email = "alice@company.com"
  name  = "Alice Johnson"
}

# Create a space
resource "arize_space" "ml_models" {
  name    = "ml-models"
  private = false
}

# Add user to space
resource "arize_space_member" "alice_access" {
  space_id = arize_space.ml_models.id
  user_id  = arize_user.alice.id
  role     = "member"
}
```

### 3. Deploy

```bash
export ARIZE_API_KEY="your-api-key-here"
terraform init
terraform plan
terraform apply
```

## Resources

| Resource | Description |
|----------|-------------|
| `arize_user` | Manage users in your Arize account |
| `arize_role` | Reference roles (custom roles must be created in Arize UI) |
| `arize_api_key` | Create and manage API keys for automation |
| `arize_space` | Create and manage ML monitoring spaces |
| `arize_space_member` | Add users to spaces with role-based access |
| `arize_saml_idp` | Configure SAML identity provider for enterprise SSO |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `arize_users` | Query all users in your account |
| `arize_roles` | Query all roles in your account (including custom roles created in UI) |
| `arize_api_keys` | Query all API keys in your account |

## Space Roles

When adding users to spaces, choose from:

- **`admin`** - Can manage space settings and add/remove members
- **`member`** - Can view all content and edit models
- **`readOnly`** - Can view content but cannot edit
- **`annotator`** - Can provide annotations and feedback

## SAML/SSO Configuration

Configure enterprise Single Sign-On (SSO) using SAML:

```hcl
resource "arize_saml_idp" "company_saml" {
  # SAML metadata URL from your identity provider
  metadata_url = "https://idp.company.com/metadata.xml"

  # Email domains allowed for this SAML IdP
  email_domains_list = ["company.com"]

  # Enforce SAML login (optional)
  enforce_saml = false

  # Automatically sync user roles from SAML attributes
  sync_user_roles = true

  # Default role for users without explicit mapping
  default_org_role_id = "member"
}

# Then assign users to spaces after SAML login
resource "arize_space_member" "team_member" {
  space_id = arize_space.ml_models.id
  user_id  = arize_user.alice.id
  role     = "admin"
}
```

**Benefits:**
- Centralized identity management (Okta, Azure AD, Google Workspace, etc.)
- Automatic user provisioning on first SAML login
- Role-based access control managed by Terraform
- Single source of truth for team structure

See [SAML SSO example](examples/06_saml_sso_setup.tf) for complete setup.

## Working with Custom Roles

Create custom roles in Arize UI, then reference them in Terraform:

```hcl
# Query all roles (including custom ones created in UI)
data "arize_roles" "all" {}

# Create a map for easy reference
locals {
  roles = { for role in data.arize_roles.all.roles : role.name => role.id }
}

# Use in SAML configuration
resource "arize_saml_idp" "sso" {
  metadata_url        = "https://idp.company.com/metadata.xml"
  email_domains_list  = ["company.com"]
  default_org_role_id = local.roles["data-scientist"]  # Reference by name
}
```

**Workflow:**
1. Create custom roles in **Arize Account Settings → Roles**
2. Use `data_arize_roles` in Terraform to query them
3. Reference by name using a local map
4. Assign to users via SAML or space membership

See [Using Existing Roles example](examples/08_using_existing_roles.tf) for advanced patterns like filtering and permission lookups.

## Import Existing Resources

Adopt Terraform for your existing Arize infrastructure:

```bash
# Import a user
terraform import arize_user.alice "VXNlcjoxMjczOTpaemlJ"

# Import a space
terraform import arize_space.prod "U3BhY2U6NTUzMzg6L25IdA=="

# Import an API key
terraform import arize_api_key.service "QXBpS2V5OjEyMzQ1Njc4OTA="
```

Then create matching Terraform configuration for the imported resource.

## Documentation & Examples

### 📖 Guides
- **[Comprehensive Guide](docs/GUIDE.md)** - Complete guide covering all use cases, advanced patterns, and best practices
- **[Architecture Guide](CLAUDE.md)** - Technical architecture, development workflow, and code patterns

### 📝 Examples
Browse practical, ready-to-use examples in the [examples/](examples/) directory:

1. **[Basic Setup](examples/01_basic_setup.tf)** - Simple user + space starter example
2. **[Team Management](examples/02_team_management.tf)** - Multi-user team with different roles
3. **[CI/CD Automation](examples/03_cicd_automation.tf)** - API keys and service accounts for automation
4. **[Multi-Environment](examples/04_multi_environment.tf)** - Dev/staging/prod with safety checks
5. **[Importing Resources](examples/05_importing_existing_resources.tf)** - Migrate existing infrastructure to Terraform
6. **[SAML/SSO Setup](examples/06_saml_sso_setup.tf)** - Enterprise SSO with role mapping and team provisioning
7. **[SAML with Custom Roles](examples/07_saml_custom_roles.tf)** - Fine-grained permission control with custom roles
8. **[Using Existing Roles](examples/08_using_existing_roles.tf)** - Query and reference roles created in Arize UI

See [examples/README.md](examples/README.md) for a detailed guide to each example.

## Building & Development

### Build the Provider

```bash
make build
```

This creates the provider binary at `bin/terraform-provider-arize`.

### Installation (Local Development)

```bash
make install
```

Installs the provider binary to your local Terraform plugin directory for testing.

### Testing

```bash
# Run unit tests
make test

# Run acceptance tests (requires ARIZE_API_KEY environment variable)
make testacc

# Format code
make fmt
```

### Development Setup

See [CLAUDE.md](CLAUDE.md) for:
- Architecture overview
- Development workflow
- Debugging tips
- Known limitations and TODOs

## Common Workflows

### Onboard a New Team Member

```bash
terraform apply -target=arize_user.new_member -target=arize_space_member.new_access
```

### Rotate API Keys

```bash
# Create new key, update services, then destroy old key
terraform destroy -target=arize_api_key.old_key
```

### Create Multiple Environments

Use the [multi-environment example](examples/04_multi_environment.tf):

```bash
terraform apply -var="environment=dev"
terraform apply -var="environment=staging"
terraform apply -var="environment=prod" -var="enable_production_space=true"
```

## Configuration Reference

### Provider Configuration

```hcl
provider "arize" {
  # API key for authentication
  # Optional: defaults to ARIZE_API_KEY environment variable
  api_key = var.arize_api_key

  # GraphQL endpoint
  # Optional: defaults to https://app.arize.com/graphql
  endpoint = "https://app.arize.com/graphql"
}
```

### Environment Variables

- `ARIZE_API_KEY` - Your Arize API key for authentication

## Requirements

- Terraform >= 1.0
- Go >= 1.21 (for building from source)
- Valid Arize account with API key

## Support

For questions, issues, or contributions, visit the [GitHub repository](https://github.com/seanlee10/terraform-provider-arize).

## License

This Terraform provider is open source and available under the [MIT License](LICENSE).

## Changelog

See [releases](https://github.com/seanlee10/terraform-provider-arize/releases) for version history and updates.
