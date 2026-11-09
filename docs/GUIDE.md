# Arize Terraform Provider - Comprehensive Guide

This guide demonstrates practical examples and use cases for managing your Arize account with Infrastructure as Code.

## Table of Contents

1. [Getting Started](#getting-started)
2. [User Management](#user-management)
3. [Space Management](#space-management)
4. [Advanced Patterns](#advanced-patterns)
5. [Best Practices](#best-practices)
6. [Common Workflows](#common-workflows)
7. [Troubleshooting](#troubleshooting)

API keys aren't managed by this provider yet; use the Arize UI or `ax api-keys`.

---

## Getting Started

### Installation

```hcl
terraform {
  required_providers {
    arize = {
      source  = "seanlee10/arize"
      version = "~> 0.3.6"
    }
  }
}

provider "arize" {
  # API key from ARIZE_API_KEY environment variable
  # or set explicitly (not recommended for production):
  # api_key  = var.arize_api_key

  # EU accounts: set the EU GraphQL endpoint. rest_endpoint (users, roles,
  # datasets, space updates and deletes, integration names) defaults to the
  # same cluster's REST API, https://api.eu-west-1a.arize.com.
  # endpoint = "https://app.eu-west-1a.arize.com/graphql"
}
```

### Required Setup

```bash
# Export your Arize API key
export ARIZE_API_KEY="your-api-key-here"

# Initialize Terraform
terraform init

# Check what will be created
terraform plan

# Apply the configuration
terraform apply
```

---

## User Management

### Creating a Single User

```hcl
resource "arize_user" "data_scientist" {
  email = "alice@company.com"
  name  = "Alice Johnson"
}

output "user_id" {
  value       = arize_user.data_scientist.id
  description = "ID of the user (an invitation ID until the invitation is accepted)"
}
```

Creating an `arize_user` sends an email invitation. Until it's accepted, the user's `status` is `invited` and the user can't be added to spaces. After acceptance, the next plan picks up the user's new ID.

### Creating Multiple Users

```hcl
locals {
  team_members = {
    "alice" = { email = "alice@company.com", name = "Alice Johnson" }
    "bob"   = { email = "bob@company.com", name = "Bob Smith" }
    "carol" = { email = "carol@company.com", name = "Carol White" }
  }
}

resource "arize_user" "team" {
  for_each = local.team_members
  
  email = each.value.email
  name  = each.value.name
}

output "team_user_ids" {
  value = {
    for name, user in arize_user.team : name => user.id
  }
}
```

### Importing Existing Users

Write the configuration for the user:

```hcl
resource "arize_user" "existing_user" {
  email = "existing@company.com"
  name  = "Existing User"
}
```

Then import it by user ID or pending invitation ID (the `arize_users` data source lists user IDs) and check the result:

```bash
terraform import arize_user.existing_user "<user_id>"
terraform state show arize_user.existing_user
terraform plan   # should show no changes
```

### Querying All Users in Account

```hcl
data "arize_users" "all" {}

locals {
  active_users = [
    for user in data.arize_users.all.users : user
    if user.status == "active"
  ]
}

output "active_user_count" {
  value = length(local.active_users)
}

output "active_users" {
  value = local.active_users
}
```

---

## Space Management

### Creating a Space for a Project

```hcl
# Create a space for production ML models
resource "arize_space" "prod_models" {
  name        = "production-models"
  private     = false
  description = "Production ML model monitoring and evaluation"
}

output "prod_space_id" {
  value = arize_space.prod_models.uuid
  description = "Space UUID for API calls"
}
```

### Creating Multiple Spaces by Environment

```hcl
locals {
  environments = {
    dev = {
      name        = "development"
      private     = true
      description = "Development environment for model testing"
    }
    staging = {
      name        = "staging"
      private     = true
      description = "Staging environment for pre-production validation"
    }
    prod = {
      name        = "production"
      private     = false
      description = "Production environment for customer-facing models"
    }
  }
}

resource "arize_space" "by_env" {
  for_each = local.environments
  
  name        = each.value.name
  private     = each.value.private
  description = each.value.description
}

output "space_ids" {
  value = {
    for env, space in arize_space.by_env : env => {
      id   = space.id
      uuid = space.uuid
    }
  }
}
```

### Adding Users to Spaces

Arize adds a user to a space only after they accept their invitation; adding a pending invitation fails with "User hasn't accepted the invitation yet". So add existing users, looked up by email:

```hcl
resource "arize_space" "ml_team" {
  name    = "ml-team-collaboration"
  private = false
}

data "arize_users" "all" {}

locals {
  user_ids = {
    for user in data.arize_users.all.users : lower(user.email) => user.id
    if user.status == "active"
  }
}

# Add users to the space with different roles
resource "arize_space_member" "alice_admin" {
  space_id = arize_space.ml_team.id
  user_id  = local.user_ids["alice@company.com"]
  role     = "admin" # Can manage space and add members
}

resource "arize_space_member" "bob_member" {
  space_id = arize_space.ml_team.id
  user_id  = local.user_ids["bob@company.com"]
  role     = "member" # Can view and edit models
}
```

Indexing `local.user_ids[...]` fails at plan time when the email isn't an active user; use `lookup(local.user_ids, email, null)` with a `precondition` for a clearer message (see [example 1](../examples/01_basic_setup/)). To invite users and add them once they accept, see [example 2](../examples/02_team_management/).

To manage a membership that already exists, import it with `terraform import arize_space_member.alice_admin "<space_id>:<user_id>"`.

### Space Roles Explained

`role` must be one of these case-sensitive values:

| Role | Access |
|------|--------|
| `admin` | Manage the space, add/remove members, configure settings |
| `member` | View all content and edit |
| `readOnly` | View content but not edit |
| `annotator` | Provide annotations and feedback |

The `annotator` role follows the user's account role: users with the Annotator account role can only be annotators in spaces, and other users can't be annotators. Other role changes are made in place.

---

## Advanced Patterns

### Complete Multi-Environment Setup

```hcl
terraform {
  required_providers {
    arize = {
      source  = "seanlee10/arize"
      version = "~> 0.3.6"
    }
  }
  
  # Use remote backend for team collaboration
  backend "s3" {
    bucket         = "my-terraform-state"
    key            = "arize/terraform.tfstate"
    region         = "us-west-2"
    encrypt        = true
    dynamodb_table = "terraform-locks"
  }
}

provider "arize" {}

# Variables for environment-specific configuration
variable "environment" {
  type    = string
  default = "dev"

  validation {
    condition     = contains(["dev", "prod"], var.environment)
    error_message = "Environment must be dev or prod."
  }
}

variable "team_emails" {
  type        = list(string)
  default     = []
  description = "Emails of existing Arize users to add to the space"
}

# Local variables
locals {
  env_config = {
    dev = {
      space_private = true
      user_roles    = "readOnly"
    }
    prod = {
      space_private = false
      user_roles    = "member"
    }
  }
  
  config = local.env_config[var.environment]
}

# Create space
resource "arize_space" "main" {
  name    = "${var.environment}-models"
  private = local.config.space_private
}

# Look up the team among active users (Arize adds users to spaces only
# after they accept their invitation)
data "arize_users" "all" {}

locals {
  active_user_ids = {
    for user in data.arize_users.all.users : lower(user.email) => user.id
    if user.status == "active"
  }
  team = {
    for email in var.team_emails : lower(email) => local.active_user_ids[lower(email)]
    if contains(keys(local.active_user_ids), lower(email))
  }
}

# Add all team members to space
resource "arize_space_member" "team_access" {
  for_each = local.team

  space_id = arize_space.main.id
  user_id  = each.value
  role     = local.config.user_roles
}

# Outputs for downstream consumption
output "space_info" {
  value = {
    id   = arize_space.main.id
    uuid = arize_space.main.uuid
    name = arize_space.main.name
  }
}

output "team_members" {
  value = keys(local.team)
}
```

### Using Data Sources to Reference Existing Resources

```hcl
# Get all users from the account
data "arize_users" "all" {}

# Find a specific user; one() returns null when there is no match
locals {
  alice = one([
    for user in data.arize_users.all.users :
    user if lower(user.email) == "alice@company.com" && user.status == "active"
  ])
}

# Add that user to a new space
resource "arize_space" "project_a" {
  name    = "project-a"
  private = false
}

resource "arize_space_member" "alice_project_a" {
  space_id = arize_space.project_a.id
  user_id  = try(local.alice.id, null)
  role     = "admin"

  lifecycle {
    precondition {
      condition     = local.alice != null
      error_message = "alice@company.com isn't an active Arize user."
    }
  }
}
```

---

## Best Practices

### 1. Use Variables for Configuration

```hcl
variable "company_domain" {
  type        = string
  default     = "company.com"
  description = "Email domain for team members"
}

# Then reference in resources
resource "arize_user" "team" {
  email = "user@${var.company_domain}"
  name  = "Team User"
}
```

### 2. Organize Code with Modules

```
terraform/
├── main.tf              # Provider and top-level config
├── spaces.tf            # Space resources
├── users.tf             # User resources
├── memberships.tf       # Space member resources
├── variables.tf         # Input variables
├── outputs.tf           # Output values
└── terraform.tfvars     # Variable values (keep out of git!)
```

### 3. Use Locals for Complex Logic

```hcl
locals {
  # Derived values
  company_users = [
    for user in data.arize_users.all.users :
    user if endswith(lower(user.email), "@company.com")
  ]
  
  # Tags/metadata
  common_labels = {
    managed_by = "terraform"
    environment = var.environment
  }
}
```

### 4. Protect Sensitive Information

Pass the Arize API key through the `ARIZE_API_KEY` environment variable (for example from your CI secret store) rather than in `.tf` files. If you pass it as a variable, mark it sensitive and keep it out of git:

```hcl
variable "arize_api_key" {
  type        = string
  sensitive   = true
  description = "Arize API key (or leave unset and use ARIZE_API_KEY)"
}

provider "arize" {
  api_key = var.arize_api_key
}

# Commit a terraform.tfvars.example without secrets,
# and add terraform.tfvars to .gitignore
```

### 5. Document Your Configuration

```hcl
# Use descriptions for all resources and variables
variable "space_name" {
  type        = string
  description = "Name of the Arize space for production models"
}

resource "arize_space" "main" {
  name        = var.space_name
  private     = false
  description = "Production space managed by Terraform - contact ML team for changes"
}
```

### 6. Import Existing Resources Before Modifying

```bash
# Before managing existing resources with Terraform:
# 1. Import them into your state
terraform import arize_user.existing "<user_id>"
terraform import arize_space.existing "<space_id>"
terraform import arize_space_member.existing "<space_id>:<user_id>"

# 2. Create matching Terraform config
# 3. Run terraform plan to verify no changes
terraform plan

# 4. Now you can manage them with Terraform
```

---

## Common Workflows

### Onboarding New Team Members

Onboarding takes two applies, because Arize adds users to spaces only after they accept their invitation.

1. Invite the user and apply; Arize emails them an invitation:

   ```hcl
   resource "arize_user" "new_member" {
     email = "newmember@company.com"
     name  = "New Member Name"
   }
   ```

2. After they accept (`terraform plan` shows their `status` change to `active` and their new ID), add the membership and apply again:

   ```hcl
   resource "arize_space_member" "new_to_prod" {
     space_id = arize_space.production.id
     user_id  = arize_user.new_member.id
     role     = "member"
   }
   ```

Applying the membership before the invitation is accepted fails with "User hasn't accepted the invitation yet"; rerun `terraform apply` after they accept. [Example 2](../examples/02_team_management/) adds members automatically once they're active.

### Creating Development & Production Environments

```hcl
# Define environments
variable "create_prod_space" {
  type    = bool
  default = false
}

# Create dev space (always)
resource "arize_space" "dev" {
  name    = "development"
  private = true
}

# Create prod space (conditionally)
resource "arize_space" "prod" {
  count   = var.create_prod_space ? 1 : 0
  name    = "production"
  private = false
}

# Run with: terraform apply -var="create_prod_space=true"
```

---

## Troubleshooting

### Issue: "User hasn't accepted the invitation yet"

**Solution:** Arize adds users to spaces only after they accept their invitation. Wait until they accept, then run `terraform apply` again.

### Issue: "Not Authorized" when creating a space

**Solution:** The API key's user can't create spaces in that organization. The error lists the user's role in each organization; account admins and users with the organization `admin` or `member` role can create spaces.

### Issue: 401 Unauthorized

**Solution:** Check the API key. Keys work only on their own cluster, so EU accounts must set `endpoint = "https://app.eu-west-1a.arize.com/graphql"`.

### Issue: Managing a space created in the Arize UI

**Solution:** Import it: `terraform import arize_space.name "<space_id>"`, then run `terraform plan` and adjust the configuration until it shows no changes.

### Issue: "User not found" when importing

**Solution:** Get the correct user ID from the `arize_users` data source, or list users directly:
```bash
# List all users to find the correct ID
curl -X POST https://app.arize.com/graphql \
  -H "x-api-key: $ARIZE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query": "query { account { users(first: 100) { edges { node { id email name } } } } }"}'
```

---

## Next Steps

- Review [CLAUDE.md](../CLAUDE.md) for architecture details
- Check [examples/](../examples/) for more configuration samples (one directory per example)
- Read the [README.md](../README.md) for installation instructions
- Explore [Terraform Registry](https://registry.terraform.io) for provider best practices
