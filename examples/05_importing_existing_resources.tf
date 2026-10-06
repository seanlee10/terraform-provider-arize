# Example 5: Importing Existing Resources & Using Data Sources
# This example demonstrates how to manage existing Arize resources that were created outside Terraform

terraform {
  required_providers {
    arize = {
      source = "arize-ai/arize"
    }
  }
}

provider "arize" {}

# ============================================================================
# STEP 1: List all existing resources using data sources
# ============================================================================

# Get all users currently in the account
data "arize_users" "all" {}

# Get all API keys
data "arize_api_keys" "all" {}

# Get all roles
data "arize_roles" "all" {}

# ============================================================================
# STEP 2: Filter existing resources
# ============================================================================

# Find users from your company domain
locals {
  company_domain = "company.com"

  company_users = [
    for user in data.arize_users.all.users :
    user if endswith(user.email, "@${local.company_domain}")
  ]

  company_user_count = length(local.company_users)

  # Group users by status
  active_users = [
    for user in local.company_users :
    user if user.status == "active"
  ]
}

# Output analysis of existing users
output "user_analysis" {
  value = {
    total_users_in_account      = length(data.arize_users.all.users)
    company_users               = local.company_user_count
    active_company_users        = length(local.active_users)
    company_users_list = [
      for user in local.company_users : {
        id    = user.id
        email = user.email
        name  = user.name
        status = user.status
      }
    ]
  }
  description = "Analysis of existing users in the Arize account"
}

# ============================================================================
# STEP 3: Import specific existing resources into Terraform
# ============================================================================

# BEFORE running terraform apply, you must import existing resources.
# Use these commands:
#
# 1. Import a specific user:
#    terraform import arize_user.existing_alice "VXNlcjoxMjczOTpaemlJ"
#
# 2. Import a specific space:
#    terraform import arize_space.existing_prod "U3BhY2U6NTUzMzg6L25IdA=="
#
# 3. Import an API key:
#    terraform import arize_api_key.existing_key "QXBpS2V5OjEyMzQ1Njc4OTA="
#
# After importing, create the Terraform configuration (see below) to manage it.

# Example: Managing an imported user
resource "arize_user" "imported_user" {
  # After running: terraform import arize_user.imported_user "user-id-here"
  # This configuration will match and manage the imported resource
  email = "existing@company.com"
  name  = "Existing User"
}

# Example: Managing an imported space
resource "arize_space" "imported_space" {
  # After running: terraform import arize_space.imported_space "space-id-here"
  name        = "existing-space"
  private     = false
  description = "Space imported from manual creation"
}

# ============================================================================
# STEP 4: Create new resources to complement existing ones
# ============================================================================

# Add newly created users to existing imported space
resource "arize_user" "new_team_member" {
  email = "new@company.com"
  name  = "New Team Member"
}

resource "arize_space_member" "new_member_access" {
  space_id = arize_space.imported_space.id
  user_id  = arize_user.new_team_member.id
  role     = "member"
}

# ============================================================================
# STEP 5: Export information about existing resources
# ============================================================================

output "existing_api_keys" {
  value = {
    count = length(data.arize_api_keys.all.api_keys)
    keys = [
      for key in data.arize_api_keys.all.api_keys : {
        id         = key.id
        name       = key.name
        created_at = key.created_at
        last_used  = key.last_used_at
      }
    ]
  }
  description = "All existing API keys in the account (do not output actual keys for security)"
}

output "existing_roles" {
  value = {
    count = length(data.arize_roles.all.roles)
    roles = [
      for role in data.arize_roles.all.roles : {
        id          = role.id
        name        = role.name
        description = role.description
      }
    ]
  }
  description = "All existing roles in the account"
}

# ============================================================================
# STEP 6: Synchronize external data with Terraform
# ============================================================================

# Find a specific existing user and use it in new configuration
locals {
  # Find user by email
  alice = [
    for user in data.arize_users.all.users :
    user if user.email == "alice@company.com"
  ]
  alice_id = length(local.alice) > 0 ? local.alice[0].id : null
}

# Use the found user in a new space
resource "arize_space" "new_collab_space" {
  name        = "collaboration-with-existing-users"
  private     = false
  description = "New space with existing team members"
}

# Add imported user to new space (if found)
resource "arize_space_member" "existing_user_in_new_space" {
  count    = local.alice_id != null ? 1 : 0
  space_id = arize_space.new_collab_space.id
  user_id  = local.alice_id
  role     = "admin"
}

# ============================================================================
# WORKFLOW: How to import existing resources
# ============================================================================

# STEP 1: Run terraform plan to see what would be created
#   terraform plan
#
# STEP 2: For each existing resource you want to manage:
#   a. Get its ID from Arize (via API or UI)
#   b. Import it: terraform import arize_<resource>.<name> <id>
#   c. Verify it's in state: terraform state show arize_<resource>.<name>
#
# STEP 3: Create the Terraform configuration matching the imported resource
#   resource "arize_<resource>" "<name>" {
#     # ... configuration ...
#   }
#
# STEP 4: Run terraform plan - should show "No changes"
#   terraform plan
#
# STEP 5: Now manage the resource through Terraform
#   terraform apply
#
# ============================================================================
# COMMON IMPORT SCENARIOS
# ============================================================================

# Scenario 1: Importing a user who was added manually
# 1. In Arize UI or via API, get user ID
# 2. terraform import arize_user.alice "VXNlcjoxMjczOTpaemlJ"
# 3. Add to config:
#    resource "arize_user" "alice" {
#      email = "alice@company.com"
#      name = "Alice"
#    }
# 4. terraform plan (should show no changes)

# Scenario 2: Importing an existing space
# 1. terraform import arize_space.production "U3BhY2U6NTUzMzg6L25IdA=="
# 2. Add to config:
#    resource "arize_space" "production" {
#      name = "production"
#      private = false
#    }
# 3. terraform plan (should show no changes)

# Scenario 3: Mix of imported and new resources
# 1. Import existing resources
# 2. Create new resources
# 3. Link them together with arize_space_member resources
# 4. Everything is now in Terraform state and can be managed together
