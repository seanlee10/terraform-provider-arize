# Example 4: Multi-Environment Setup - Dev, Staging, Production
#
# One configuration for every environment; the environment picks the space
# settings and the role the team gets. Use a separate state per environment
# (a workspace or a backend key per environment).
#
#   terraform apply -var="environment=dev"
#   terraform apply -var="environment=prod" -var="enable_production=true"
#   terraform apply -var-file=prod.tfvars   # see prod.tfvars.example

terraform {
  # Variable validation that refers to another variable needs Terraform 1.9+
  required_version = ">= 1.9"

  required_providers {
    arize = {
      source  = "seanlee10/arize"
      version = "~> 0.3.6"
    }
  }

  # In production, use a remote state backend:
  # backend "s3" {
  #   bucket         = "my-terraform-state"
  #   key            = "arize/terraform.tfstate"
  #   region         = "us-west-2"
  #   encrypt        = true
  #   dynamodb_table = "terraform-locks"
  # }
}

provider "arize" {}

variable "environment" {
  type        = string
  default     = "dev"
  description = "Environment name: dev, staging, or prod"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }
}

# Safety check: plan fails for prod unless this is set explicitly.
variable "enable_production" {
  type        = bool
  default     = false
  description = "Must be true to plan or apply the prod environment"

  validation {
    condition     = var.environment != "prod" || var.enable_production
    error_message = "Set enable_production=true to manage the prod environment."
  }
}

variable "team_emails" {
  type        = list(string)
  default     = []
  description = "Emails of existing Arize users to add to this environment's space"
}

locals {
  environment_config = {
    dev = {
      space_name     = "development-models"
      space_private  = true
      team_role      = "member"
      backup_enabled = false
    }
    staging = {
      space_name     = "staging-models"
      space_private  = true
      team_role      = "readOnly"
      backup_enabled = true
    }
    prod = {
      space_name     = "production-models"
      space_private  = false
      team_role      = "readOnly"
      backup_enabled = true
    }
  }

  config = local.environment_config[var.environment]
}

resource "arize_space" "main" {
  name        = local.config.space_name
  private     = local.config.space_private
  description = "${var.environment} space for model monitoring and evaluation"
}

# Arize adds users to spaces only after they accept their invitation, so the
# team is looked up among active users. Unknown emails are listed in the
# missing_team_members output instead of failing the apply.
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

resource "arize_space_member" "team_access" {
  for_each = local.team

  space_id = arize_space.main.id
  user_id  = each.value
  role     = local.config.team_role
}

# Backup space for staging and prod
resource "arize_space" "backup" {
  count       = local.config.backup_enabled ? 1 : 0
  name        = "${var.environment}-backup"
  private     = true
  description = "Backup space for ${var.environment}"
}

output "environment_info" {
  value = {
    name          = var.environment
    space_id      = arize_space.main.id
    space_uuid    = arize_space.main.uuid
    space_name    = arize_space.main.name
    space_private = arize_space.main.private
  }
  description = "Key information about the environment"
}

output "team_members" {
  value       = keys(local.team)
  description = "Users added to the environment's space"
}

output "missing_team_members" {
  value       = [for email in var.team_emails : email if !contains(keys(local.team), lower(email))]
  description = "Emails in team_emails that aren't active Arize users yet"
}

output "backup_space_id" {
  value       = try(arize_space.backup[0].id, null)
  description = "Backup space ID (null if backups are not enabled)"
}
