# Example 8: Using Existing Roles (Created in Arize UI)
# This example demonstrates how to read and reference roles created in the Arize UI
# without needing to create them via Terraform

terraform {
  required_providers {
    arize = {
      source = "arize-ai/arize"
    }
  }
}

provider "arize" {}

# ============================================================================
# STEP 1: Read All Roles from Arize
# ============================================================================

# Query all roles that exist in your Arize account
data "arize_roles" "all" {}

# ============================================================================
# STEP 2: Create a Map of Role Names to IDs (for easy reference)
# ============================================================================

locals {
  # Create a map: role_name => role_id
  # This allows you to reference roles by name instead of ID
  role_ids = {
    for role in data.arize_roles.all.roles :
    role.name => role.id
  }

  # Example: Find a specific role
  data_scientist_role_id = local.role_ids["data-scientist"]
  analyst_role_id        = local.role_ids["analyst"]
  admin_role_id          = local.role_ids["admin"]
}

# ============================================================================
# STEP 3: Use Data Source to Filter Roles
# ============================================================================

locals {
  # Find all custom roles (non-predefined)
  custom_roles = [
    for role in data.arize_roles.all.roles :
    role if !contains(["admin", "member", "readOnly", "annotator"], role.name)
  ]

  # Find roles with specific permissions
  roles_with_write_permission = [
    for role in data.arize_roles.all.roles :
    role if contains(role.permissions, "write:models")
  ]
}

# ============================================================================
# STEP 4: Reference Roles in SAML Configuration
# ============================================================================

resource "arize_saml_idp" "company_saml" {
  metadata_url = "https://idp.company.com/metadata.xml"

  email_domains_list = ["company.com"]

  # Use the data scientist role created in Arize UI
  default_org_role_id = local.data_scientist_role_id

  enforce_saml    = false
  sync_user_roles = true
}

# ============================================================================
# STEP 5: Reference Roles in Space Assignments
# ============================================================================

# Create a space and assign users (they'll get custom roles via SAML)
resource "arize_space" "data_science" {
  name        = "data-science"
  private     = true
  description = "Data science team with custom role permissions"
}

resource "arize_user" "alice" {
  email = "alice@company.com"
  name  = "Alice - Data Scientist"
}

resource "arize_space_member" "alice_admin" {
  space_id = arize_space.data_science.id
  user_id  = arize_user.alice.id
  role     = "admin"  # Space role
}

# ============================================================================
# OUTPUTS: Display All Available Roles
# ============================================================================

output "all_roles" {
  value = {
    for role in data.arize_roles.all.roles :
    role.name => {
      id          = role.id
      description = role.description
      permissions = role.permissions
    }
  }
  description = "All roles available in your Arize account"
}

output "role_ids_map" {
  value       = local.role_ids
  description = "Mapping of role names to IDs for easy reference"
}

output "custom_roles" {
  value = {
    for role in local.custom_roles :
    role.name => {
      id          = role.id
      permissions = role.permissions
    }
  }
  description = "Custom roles (not predefined)"
}

output "roles_with_write_permission" {
  value = {
    for role in local.roles_with_write_permission :
    role.name => role.id
  }
  description = "Roles that have write:models permission"
}

# ============================================================================
# WORKFLOW: How to Use Existing Roles
# ============================================================================
#
# 1. Create custom roles in Arize UI:
#    - Go to Account Settings → Roles
#    - Click "Create Role"
#    - Name: "data-scientist"
#    - Select permissions you want
#    - Save
#
# 2. In Terraform:
#    - Use data "arize_roles" "all" {} to fetch all roles
#    - Create a local map: role_ids = { for role in ... }
#    - Reference by name: local.role_ids["data-scientist"]
#    - Use in SAML, space assignments, etc.
#
# 3. Benefits:
#    - No hardcoded role IDs
#    - Easy to reference by role name
#    - Automatic discovery of new roles
#    - Works with roles created in UI
#
# ============================================================================
# EXAMPLE: Finding a Role by Name
# ============================================================================
#
# data "arize_roles" "all" {}
#
# locals {
#   # Find exact role by name
#   my_role = [
#     for role in data.arize_roles.all.roles :
#     role if role.name == "data-scientist"
#   ][0]
# }
#
# resource "arize_saml_idp" "sso" {
#   metadata_url        = "..."
#   email_domains_list  = ["company.com"]
#   default_org_role_id = local.my_role.id
# }
#
# ============================================================================
# BEST PRACTICES
# ============================================================================
#
# ✅ Use data source to read roles created in UI
# ✅ Create a local map for easy reference by name
# ✅ Filter roles using for expressions for complex queries
# ✅ Reference role IDs in SAML, space assignments, etc.
#
# ❌ Don't hardcode role IDs - use the data source instead
# ❌ Don't assume role names - filter and check existence first
# ❌ Don't rely on role order - use name-based lookups
#
# ============================================================================
