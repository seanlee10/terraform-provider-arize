# Example 4: Multi-Environment Setup - Dev, Staging, Production
# This example demonstrates managing multiple environments with consistent configuration

terraform {
  required_providers {
    arize = {
      source = "arize-ai/arize"
    }
  }

  # In production, use remote state backend:
  # backend "s3" {
  #   bucket         = "my-terraform-state"
  #   key            = "arize/terraform.tfstate"
  #   region         = "us-west-2"
  #   encrypt        = true
  #   dynamodb_table = "terraform-locks"
  # }
}

provider "arize" {}

# Input variables for environment-specific configuration
variable "environment" {
  type        = string
  default     = "dev"
  description = "Environment name: dev, staging, or prod"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }
}

variable "team_size" {
  type        = number
  default     = 3
  description = "Number of team members to create"
}

variable "enable_production_space" {
  type        = bool
  default     = false
  description = "Whether to create production space (safety check)"
}

# Environment-specific configuration
locals {
  environment_config = {
    dev = {
      space_private   = true
      space_name      = "development-models"
      default_role    = "member"
      api_key_desc    = "Development API key"
      backup_enabled  = false
    }
    staging = {
      space_private   = true
      space_name      = "staging-models"
      default_role    = "readOnly"
      api_key_desc    = "Staging API key (read-only)"
      backup_enabled  = true
    }
    prod = {
      space_private   = false
      space_name      = "production-models"
      default_role    = "readOnly"
      api_key_desc    = "Production API key"
      backup_enabled  = true
    }
  }

  config = local.environment_config[var.environment]

  common_tags = {
    managed_by  = "terraform"
    environment = var.environment
    created_at  = timestamp()
  }
}

# Guard against accidentally creating production without explicit flag
resource "null_resource" "production_safety_check" {
  count = var.environment == "prod" && !var.enable_production_space ? 1 : 0

  provisioner "local-exec" {
    command = "echo 'ERROR: Set enable_production_space=true to create production resources' && exit 1"
  }
}

# Create environment-specific space
resource "arize_space" "main" {
  name        = local.config.space_name
  private     = local.config.space_private
  description = "${var.environment} space for model monitoring and evaluation"
}

# Create team members for this environment
resource "arize_user" "team" {
  count = var.team_size

  email = "${var.environment}-user${count.index}@company.com"
  name  = "Team Member ${count.index + 1} (${var.environment})"
}

# Add team members to space with environment-specific roles
resource "arize_space_member" "team_access" {
  count    = length(arize_user.team)
  space_id = arize_space.main.id
  user_id  = arize_user.team[count.index].id
  role     = local.config.default_role
}

# Create service account for this environment
resource "arize_user" "service_account" {
  email = "${var.environment}-service@company.com"
  name  = "${var.environment} Service Account"
}

# Add service account with read-only access
resource "arize_space_member" "service_account" {
  space_id = arize_space.main.id
  user_id  = arize_user.service_account.id
  role     = "member"
}

# Create API key for environment
resource "arize_api_key" "environment" {
  name        = "${var.environment}-api-key"
  permissions = var.environment == "prod" ? ["api:read"] : ["api:read", "api:write"]
}

# Create backup space if enabled
resource "arize_space" "backup" {
  count       = local.config.backup_enabled ? 1 : 0
  name        = "${var.environment}-backup"
  private     = true
  description = "Backup space for ${var.environment} - data safety"
}

# Outputs for this environment
output "environment_info" {
  value = {
    name        = var.environment
    space_id    = arize_space.main.id
    space_uuid  = arize_space.main.uuid
    space_name  = arize_space.main.name
    space_private = arize_space.main.private
    api_key_id  = arize_api_key.environment.id
  }
  description = "Key information about the ${var.environment} environment"
}

output "team_info" {
  value = {
    count = length(arize_user.team)
    members = [
      for i, user in arize_user.team : {
        id    = user.id
        email = user.email
        name  = user.name
      }
    ]
  }
  description = "Team members in ${var.environment}"
}

output "service_account_id" {
  value = arize_user.service_account.id
  description = "Service account ID for ${var.environment}"
}

output "backup_space_id" {
  value = try(arize_space.backup[0].id, null)
  description = "Backup space ID (null if backups not enabled)"
}

# Useful Terraform commands for this setup:
#
# Deploy to development:
#   terraform plan -var="environment=dev"
#   terraform apply -var="environment=dev"
#
# Deploy to staging:
#   terraform plan -var="environment=staging"
#   terraform apply -var="environment=staging"
#
# Deploy to production (requires explicit flag):
#   terraform plan -var="environment=prod" -var="enable_production_space=true"
#   terraform apply -var="environment=prod" -var="enable_production_space=true"
#
# Use terraform.tfvars for environment-specific defaults:
#   environment = "prod"
#   enable_production_space = true
#   team_size = 5
