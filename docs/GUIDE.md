# Arize Terraform Provider - Comprehensive Guide

This guide demonstrates practical examples and use cases for managing your Arize account with Infrastructure as Code.

## Table of Contents

1. [Getting Started](#getting-started)
2. [User Management](#user-management)
3. [Space Management](#space-management)
4. [API Keys & Automation](#api-keys--automation)
5. [Advanced Patterns](#advanced-patterns)
6. [Best Practices](#best-practices)

---

## Getting Started

### Installation

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
  # or set explicitly (not recommended for production):
  # api_key  = var.arize_api_key
  
  # Optional: custom endpoint for self-hosted instances
  # endpoint = "https://arize-app.my-domain.com/graphql"
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
  value = arize_user.data_scientist.id
  description = "ID of the created user"
}
```

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

```bash
# First, find the user ID from Arize
# Then import it into Terraform

terraform import arize_user.existing_user "VXNlcjoxMjczOTpaemlJ"
```

Then reference it in your config:

```hcl
resource "arize_user" "existing_user" {
  email = "existing@company.com"
  name  = "Existing User"
}

# Now Terraform will manage this user
terraform state show arize_user.existing_user
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

```hcl
# Create a space
resource "arize_space" "ml_team" {
  name    = "ml-team-collaboration"
  private = false
}

# Create team members
resource "arize_user" "alice" {
  email = "alice@company.com"
  name  = "Alice"
}

resource "arize_user" "bob" {
  email = "bob@company.com"
  name  = "Bob"
}

# Add users to the space with different roles
resource "arize_space_member" "alice_admin" {
  space_id = arize_space.ml_team.id
  user_id  = arize_user.alice.id
  role     = "admin"  # Can manage space and add members
}

resource "arize_space_member" "bob_member" {
  space_id = arize_space.ml_team.id
  user_id  = arize_user.bob.id
  role     = "member"  # Can view and edit models
}
```

### Space Roles Explained

```hcl
# admin - Can manage space, add/remove members, configure settings
resource "arize_space_member" "admin_user" {
  space_id = arize_space.shared.id
  user_id  = arize_user.manager.id
  role     = "admin"
}

# member - Can view all content and edit models
resource "arize_space_member" "editor_user" {
  space_id = arize_space.shared.id
  user_id  = arize_user.engineer.id
  role     = "member"
}

# readOnly - Can view content but cannot edit
resource "arize_space_member" "viewer_user" {
  space_id = arize_space.shared.id
  user_id  = arize_user.stakeholder.id
  role     = "readOnly"
}

# annotator - Can provide annotations/feedback on predictions
resource "arize_space_member" "labeler_user" {
  space_id = arize_space.shared.id
  user_id  = arize_user.labeler.id
  role     = "annotator"
}
```

---

## API Keys & Automation

### Creating an API Key for CI/CD

```hcl
resource "arize_api_key" "deployment_pipeline" {
  name        = "deployment-pipeline-api-key"
  permissions = [
    "api:read",
    "api:write"
  ]
  expires_at = "2027-12-31T23:59:59Z"  # Optional expiration
}

# Output the key (only available at creation!)
output "deployment_api_key" {
  value       = arize_api_key.deployment_pipeline.key
  sensitive   = true
  description = "Store this in your CI/CD secret manager"
}
```

### Creating Multiple API Keys for Different Services

```hcl
locals {
  services = {
    data_ingestion = {
      name        = "data-ingestion-service"
      permissions = ["api:write"]
    }
    monitoring = {
      name        = "monitoring-service"
      permissions = ["api:read"]
    }
    exports = {
      name        = "export-service"
      permissions = ["api:read"]
    }
  }
}

resource "arize_api_key" "services" {
  for_each    = local.services
  name        = each.value.name
  permissions = each.value.permissions
}

# Save keys to a secure location (e.g., AWS Secrets Manager)
output "service_api_keys" {
  value = {
    for service, key in arize_api_key.services : service => {
      id   = key.id
      name = key.name
      # DO NOT output the actual key - store it separately!
    }
  }
}
```

### Querying Existing API Keys

```hcl
data "arize_api_keys" "all" {}

# Find old API keys that should be rotated
locals {
  thirty_days_ago = timestamp() - 30 * 24 * 60 * 60
  
  old_keys = [
    for key in data.arize_api_keys.all.api_keys :
    key if timeadd(key.created_at, "720h") < timestamp()  # Older than 30 days
  ]
}

output "keys_needing_rotation" {
  value = [
    for key in local.old_keys : {
      id         = key.id
      name       = key.name
      created_at = key.created_at
    }
  ]
}
```

---

## Advanced Patterns

### Complete Multi-Environment Setup

```hcl
terraform {
  required_providers {
    arize = {
      source  = "arize-ai/arize"
      version = "~> 0.0.1"
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
}

variable "team_size" {
  type    = number
  default = 3
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

# Create team members from a list
resource "arize_user" "team" {
  count = var.team_size
  
  email = "user${count.index}@company.com"
  name  = "Team Member ${count.index + 1}"
}

# Add all team members to space
resource "arize_space_member" "team_access" {
  count    = length(arize_user.team)
  space_id = arize_space.main.id
  user_id  = arize_user.team[count.index].id
  role     = local.config.user_roles
}

# Create service account API key
resource "arize_api_key" "service" {
  name        = "${var.environment}-service-account"
  permissions = ["api:read", "api:write"]
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
  value = [
    for user in arize_user.team : {
      id    = user.id
      email = user.email
      name  = user.name
    }
  ]
}
```

### Using Data Sources to Reference Existing Resources

```hcl
# Get all users from the account
data "arize_users" "all" {}

# Filter to find a specific user
locals {
  alice = [
    for user in data.arize_users.all.users :
    user if user.email == "alice@company.com"
  ][0]
}

# Add that user to a new space
resource "arize_space" "project_a" {
  name    = "project-a"
  private = false
}

resource "arize_space_member" "alice_project_a" {
  space_id = arize_space.project_a.id
  user_id  = local.alice.id
  role     = "admin"
}
```

---

## Best Practices

### 1. Use Variables for Configuration

```hcl
variable "arize_api_key" {
  type        = string
  sensitive   = true
  description = "Arize API Key (use env var ARIZE_API_KEY)"
}

variable "company_domain" {
  type        = string
  default     = "company.com"
  description = "Email domain for team members"
}

# Then reference in resources
resource "arize_user" "team" {
  email = "user@${var.company_domain}"
}
```

### 2. Organize Code with Modules

```
terraform/
├── main.tf              # Provider and top-level config
├── spaces.tf            # Space resources
├── users.tf             # User resources
├── api_keys.tf          # API key resources
├── variables.tf         # Input variables
├── outputs.tf           # Output values
└── terraform.tfvars     # Variable values (keep out of git!)
```

### 3. Use Locals for Complex Logic

```hcl
locals {
  # Derived values
  production_users = [
    for user in data.arize_users.all.users :
    user if contains(user.email, "company.com")
  ]
  
  # Tags/metadata
  common_labels = {
    managed_by = "terraform"
    environment = var.environment
  }
}
```

### 4. Protect Sensitive Information

```hcl
# Store API keys securely
resource "arize_api_key" "sensitive" {
  name        = "sensitive-key"
  permissions = ["api:read", "api:write"]
}

# Mark output as sensitive
output "api_key" {
  value     = arize_api_key.sensitive.key
  sensitive = true
}

# Use tfvars.example (no actual secrets)
# But add terraform.tfvars to .gitignore
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
terraform import arize_user.existing "user-id-here"
terraform import arize_space.existing "space-id-here"

# 2. Create matching Terraform config
# 3. Run terraform plan to verify no changes
terraform plan

# 4. Now you can manage them with Terraform
```

---

## Common Workflows

### Onboarding New Team Members

```hcl
# Create new user
resource "arize_user" "new_member" {
  email = "newmember@company.com"
  name  = "New Member Name"
}

# Add to production space
resource "arize_space_member" "new_to_prod" {
  space_id = arize_space.production.id
  user_id  = arize_user.new_member.id
  role     = "member"
}

# Terraform plan and apply
# → User created in Arize
# → User added to production space automatically
```

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

### Rotating API Keys

```bash
# Create new key
resource "arize_api_key" "new_v2" {
  name        = "deployment-key-v2"
  permissions = ["api:read", "api:write"]
}

# Update your CI/CD system with new key from output
terraform output deployment_api_key_new

# Once confirmed working, remove old key
# (Delete the old resource from your Terraform config)

terraform destroy -target=arize_api_key.old_v1
```

---

## Troubleshooting

### Issue: "Space already exists in account"

**Solution:** A space with that name already exists. Either:
1. Import it: `terraform import arize_space.name "space-id"`
2. Use a unique name in your configuration

### Issue: "API key is sensitive - cannot output"

**Solution:** Mark outputs as sensitive:
```hcl
output "key" {
  value     = arize_api_key.service.key
  sensitive = true  # This hides the value from logs
}
```

### Issue: "User not found" when importing

**Solution:** Get the correct user ID:
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
- Check [examples/](../examples/) for more configuration samples
- Read the [README.md](../README.md) for installation instructions
- Explore [Terraform Registry](https://registry.terraform.io) for provider best practices
