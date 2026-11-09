# Example 8: Using Existing Roles (Created in Arize UI)
# This example demonstrates how to read and reference roles created in the Arize UI
# without needing to create them via Terraform

terraform {
  required_providers {
    arize = {
      source  = "seanlee10/arize"
      version = "~> 0.3.6"
    }
  }
}

provider "arize" {}

variable "org_id" {
  description = "Arize organization that SAML users join (Settings -> Organizations)"
  type        = string
}

variable "data_scientist_role_name" {
  description = "Name of a custom role created in the Arize UI"
  type        = string
  default     = "data-scientist"
}

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

  # Look up a role by name. lookup() returns null when the account has no
  # such role; the precondition on arize_saml_idp turns that into a clear error.
  data_scientist_role_id = lookup(local.role_ids, var.data_scientist_role_name, null)
}

# ============================================================================
# STEP 3: Use Data Source to Filter Roles
# ============================================================================

locals {
  # Find all custom roles (non-predefined)
  custom_roles = [
    for role in data.arize_roles.all.roles :
    role if !role.is_predefined
  ]

  # Find roles with specific permissions
  roles_with_write_permission = [
    for role in data.arize_roles.all.roles :
    role if contains(role.permissions, "DATASET_CREATE")
  ]
}

# ============================================================================
# STEP 4: Reference Roles in SAML Configuration
# ============================================================================

resource "arize_saml_idp" "company_saml" {
  metadata_url = "https://idp.company.com/metadata.xml"

  email_domains_list = ["company.com"]

  enforce_saml    = false
  sync_user_roles = true

  # Give the IdP's data-science group the custom role created in the Arize UI.
  # Custom roles apply per space; org_role must be a built-in name.
  role_mappings = [{
    attributes         = { groups = "data-science" }
    org_id             = var.org_id
    org_role           = "member"
    space_custom_roles = { (arize_space.data_science.id) = local.data_scientist_role_id }
  }]

  lifecycle {
    precondition {
      condition     = local.data_scientist_role_id != null
      error_message = "No role named \"${var.data_scientist_role_name}\" exists in Arize. Create it in the UI or set data_scientist_role_name."
    }
  }
}

# ============================================================================
# STEP 5: The Space the Custom Role Applies To
# ============================================================================

# Members of the IdP's data-science group get the custom role in this space
# on their SAML login. (arize_space_member takes only the built-in space
# roles: admin, member, readOnly, annotator.)
resource "arize_space" "data_science" {
  name        = "data-science"
  private     = true
  description = "Data science team with custom role permissions"
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
  description = "Roles that have the DATASET_CREATE permission"
}

# ============================================================================
# WORKFLOW: How to Use Existing Roles
# ============================================================================
#
# 1. Create custom roles in Arize UI (Account Settings → Roles), or manage
#    them with the arize_role resource (see example 7)
#
# 2. In Terraform:
#    - Use data "arize_roles" "all" {} to fetch all roles
#    - Create a local map: role_ids = { for role in ... }
#    - Reference by name: lookup(local.role_ids, "data-scientist", null),
#      with a precondition so a missing role fails with a clear message
#    - Use in SAML role_mappings (space_custom_roles, project_roles)
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
#   # Find exact role by name; one() returns null if there is none
#   my_role = one([
#     for role in data.arize_roles.all.roles :
#     role if role.name == "data-scientist"
#   ])
# }
#
# resource "arize_saml_idp" "sso" {
#   metadata_url       = "..."
#   email_domains_list = ["company.com"]
#   role_mappings = [{
#     attributes         = { groups = "data-science" }
#     org_id             = var.org_id
#     org_role           = "member"
#     space_custom_roles = { (arize_space.data_science.id) = try(local.my_role.id, null) }
#   }]
#   lifecycle {
#     precondition {
#       condition     = local.my_role != null
#       error_message = "Role data-scientist not found."
#     }
#   }
# }
#
# ============================================================================
# BEST PRACTICES
# ============================================================================
#
# ✅ Use data source to read roles created in UI
# ✅ Create a local map for easy reference by name
# ✅ Filter roles using for expressions for complex queries
# ✅ Reference role IDs in SAML role mappings
#
# ❌ Don't hardcode role IDs - use the data source instead
# ❌ Don't assume role names - filter and check existence first
# ❌ Don't rely on role order - use name-based lookups
#
# ============================================================================
