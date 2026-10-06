# Terraform Provider for Arize

Manage Arize resources (users, roles, API keys) via Terraform using GraphQL API.

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
make testacc     # Acceptance tests (requires ARIZE_API_KEY)
```

## Resources

- `arize_user` - Manage users
- `arize_role` - Manage roles
- `arize_api_key` - Manage API keys

## Data Sources

- `arize_users` - List users
- `arize_roles` - List roles
- `arize_api_keys` - List API keys

## Import

Import existing resources:

```bash
terraform import arize_user.example <user-id>
terraform import arize_role.example <role-id>
terraform import arize_api_key.example <api-key-id>
```
