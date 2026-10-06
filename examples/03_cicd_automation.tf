# Example 3: CI/CD & Automation - API Keys and Service Accounts
# This example demonstrates setting up API keys for automated workflows

terraform {
  required_providers {
    arize = {
      source = "arize-ai/arize"
    }
  }
}

provider "arize" {}

# Define services and their API permissions
locals {
  services = {
    data_pipeline = {
      description = "Data ingestion and model evaluation pipeline"
      permissions = ["api:write"]  # Only write data
    }

    monitoring_service = {
      description = "Model monitoring and alerting service"
      permissions = ["api:read"]  # Only read metrics
    }

    export_service = {
      description = "Data export to data warehouse"
      permissions = ["api:read"]  # Only read data
    }

    full_access = {
      description = "Full access for internal tools"
      permissions = ["api:read", "api:write"]
    }
  }
}

# Create API keys for each service
resource "arize_api_key" "services" {
  for_each = local.services

  name        = "${each.key}-api-key"
  permissions = each.value.permissions

  # Optional: Set expiration date (30 days from now)
  # expires_at = timeadd(timestamp(), "720h")
}

# Sensitive output - these keys should be stored in a secret manager
output "service_api_keys" {
  value = {
    for service, key in arize_api_key.services : service => {
      id   = key.id
      name = key.name
      # The actual key is marked sensitive and won't be printed
    }
  }
  description = "API key IDs and names (actual keys are sensitive)"
}

# This outputs the actual keys - only run once and store securely!
output "api_keys_secret_export" {
  value = {
    for service, key in arize_api_key.services : service => {
      key = key.key
      description = local.services[service].description
    }
  }
  sensitive = true
  description = "API keys for each service - STORE SECURELY IN SECRET MANAGER"
}

# Example: Save to AWS Secrets Manager (requires aws provider)
# resource "aws_secretsmanager_secret" "arize_api_keys" {
#   for_each = arize_api_key.services
#   name     = "arize/${each.key}/api-key"
# }
#
# resource "aws_secretsmanager_secret_version" "arize_api_keys" {
#   for_each      = arize_api_key.services
#   secret_id     = aws_secretsmanager_secret.arize_api_keys[each.key].id
#   secret_string = each.value.key
# }

# List all existing API keys in the account
data "arize_api_keys" "all" {}

output "all_api_keys_count" {
  value = length(data.arize_api_keys.all.api_keys)
  description = "Total number of API keys in the account"
}

output "api_keys_created_today" {
  value = [
    for key in data.arize_api_keys.all.api_keys :
    {
      id   = key.id
      name = key.name
      created_at = key.created_at
    }
    if timeadd(key.created_at, "0s") > timestamp() - 86400
  ]
  description = "API keys created in the last 24 hours"
}

# Example: Identify API keys that need rotation
locals {
  rotation_age_days = 90
  keys_needing_rotation = [
    for key in data.arize_api_keys.all.api_keys :
    {
      id         = key.id
      name       = key.name
      created_at = key.created_at
      age_days   = floor((timestamp() - timeadd(key.created_at, "0s")) / 86400)
    }
    if floor((timestamp() - timeadd(key.created_at, "0s")) / 86400) > local.rotation_age_days
  ]
}

output "keys_needing_rotation" {
  value = local.keys_needing_rotation
  description = "API keys older than ${local.rotation_age_days} days that should be rotated"
}
