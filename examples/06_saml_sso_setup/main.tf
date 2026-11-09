# Example 6: SAML SSO Setup with Role Mapping
# This example demonstrates how to configure SAML/SSO for Arize with automatic role mapping

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

# ============================================================================
# STEP 1: Configure SAML Identity Provider
# ============================================================================

# Create a SAML IdP configuration
resource "arize_saml_idp" "company_saml" {
  # Option 1: Use metadata URL (preferred)
  metadata_url = "https://idp.company.com/metadata.xml"

  # Option 2: Use raw metadata XML (if metadata_url not available)
  # metadata_xml = file("${path.module}/saml-metadata.xml")

  # Email domains allowed for this IdP
  email_domains_list = ["company.com", "subsidiary.company.com"]

  # Enforce SAML login - when true, users can only login via SAML
  enforce_saml = false

  # Re-apply role mappings on every login, so IdP group changes take effect
  sync_user_roles = true

  # Map IdP groups to Arize roles. Org and space roles here are built-in
  # names: admin, member, readOnly, annotator.
  role_mappings = [
    {
      attributes  = { groups = "arize-data-science" }
      org_id      = var.org_id
      org_role    = "member"
      space_roles = { (arize_space.data_science.id) = "admin" }
    },
    {
      attributes  = { groups = "arize-ml-ops" }
      org_id      = var.org_id
      org_role    = "member"
      space_roles = { (arize_space.ml_ops.id) = "member" }
    },
    {
      attributes  = { groups = "arize-analytics" }
      org_id      = var.org_id
      org_role    = "readOnly"
      space_roles = { (arize_space.analytics.id) = "readOnly" }
    },
    {
      attributes       = { groups = "arize-admins" }
      is_account_admin = true
    },
  ]

  # Users in none of the groups above are rejected. To let them in instead,
  # set allow_login_with_defaults = true plus all four default_* attributes:
  #   default_org_id        = var.org_id
  #   default_org_role_id   = "readOnly"
  #   default_space_id      = arize_space.analytics.id
  #   default_space_role_id = "readOnly"
}

# ============================================================================
# STEP 2: Create spaces for different teams
# ============================================================================

resource "arize_space" "data_science" {
  name        = "data-science"
  private     = true
  description = "Data Science team space"
}

resource "arize_space" "ml_ops" {
  name        = "ml-ops"
  private     = true
  description = "ML Operations team space"
}

resource "arize_space" "analytics" {
  name        = "analytics"
  private     = false
  description = "Analytics team space"
}

# Users don't need arize_user or arize_space_member resources: Arize creates
# them on their first SAML login and gives them the org and space roles of the
# matching role_mappings entry. (arize_space_member can only add users who
# already have an account, and only after they accepted their invitation.)

# ============================================================================
# OUTPUTS
# ============================================================================

output "saml_idp_id" {
  value       = arize_saml_idp.company_saml.id
  description = "SAML IdP configuration ID"
}

output "spaces" {
  value = {
    data_science = arize_space.data_science.id
    ml_ops       = arize_space.ml_ops.id
    analytics    = arize_space.analytics.id
  }
  description = "Space IDs for team collaboration"
}

# ============================================================================
# WORKFLOW: How SAML SSO Works
# ============================================================================
#
# 1. Configure SAML IdP with metadata URL or XML
#    - Set email domains that are allowed
#    - Map IdP groups (SAML attributes) to Arize roles with role_mappings
#    - Enable sync_user_roles to re-apply mappings on every login
#
# 2. First SAML login:
#    - User logs in through your identity provider (Okta, Azure AD, etc)
#    - Arize receives SAML assertion with user attributes
#    - User is auto-created in Arize with the roles of the matching mapping
#
# 3. Role mapping:
#    - role_mappings: per-group org, space, and project roles
#    - allow_login_with_defaults + default_*: one role set for everyone else
#    - Destroying arize_saml_idp only removes it from state; the Arize API
#      cannot delete an IdP, so delete it in the Arize UI
#
# 4. Subsequent logins:
#    - With sync_user_roles = true, roles are re-applied from the IdP groups
#
# ============================================================================
# ADVANTAGES OF THIS SETUP
# ============================================================================
#
# ✅ Centralized identity management via SAML/SSO
# ✅ Automatic user provisioning on first login
# ✅ Terraform-managed team structure and permissions
# ✅ Audit trail through terraform state
# ✅ Easy offboarding (remove the user from the IdP)
# ✅ Consistent role assignment across teams
#
# ============================================================================
# COMMON WORKFLOWS
# ============================================================================
#
# Onboard a new team member:
#   1. Add them to the right group in your identity provider (Okta, Azure, etc)
#   2. They log in via SAML and get that group's roles
#
# Offboard a team member:
#   1. Remove them from your identity provider
#   2. If they were managed with arize_user, remove it and terraform apply
#      (this deactivates the user in Arize)
#
# Add a new team/space:
#   1. Create a new arize_space resource
#   2. Add a role_mappings entry for the team's IdP group
#   3. terraform apply
