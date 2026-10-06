# Example 6: SAML SSO Setup with Role Mapping
# This example demonstrates how to configure SAML/SSO for Arize with automatic role mapping

terraform {
  required_providers {
    arize = {
      source = "arize-ai/arize"
    }
  }
}

provider "arize" {}

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

  # Automatically sync user roles from SAML attributes
  sync_user_roles = true

  # Default role for users without explicit mapping
  default_org_role_id = "admin"

  # Allow login with defaults if no specific mapping exists
  allow_login_with_defaults = true
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

# ============================================================================
# STEP 3: Create users (optional - will be auto-created on first SAML login)
# ============================================================================

resource "arize_user" "data_scientist" {
  email = "alice@company.com"
  name  = "Alice Johnson"
}

resource "arize_user" "ml_engineer" {
  email = "bob@company.com"
  name  = "Bob Smith"
}

resource "arize_user" "analyst" {
  email = "charlie@company.com"
  name  = "Charlie Davis"
}

# ============================================================================
# STEP 4: Assign users to spaces with roles based on SAML attributes
# ============================================================================

# Data scientists - admin in data science space
resource "arize_space_member" "data_scientist_space" {
  space_id = arize_space.data_science.id
  user_id  = arize_user.data_scientist.id
  role     = "admin"
}

# ML engineers - member in ml ops space
resource "arize_space_member" "ml_engineer_space" {
  space_id = arize_space.ml_ops.id
  user_id  = arize_user.ml_engineer.id
  role     = "member"
}

# Analysts - read-only in analytics space
resource "arize_space_member" "analyst_space" {
  space_id = arize_space.analytics.id
  user_id  = arize_user.analyst.id
  role     = "readOnly"
}

# ============================================================================
# STEP 5: Create API keys for service accounts
# ============================================================================

# Service account for automated monitoring
resource "arize_api_key" "monitoring_service" {
  name        = "monitoring-automation"
  permissions = ["read:models", "write:predictions"]
}

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

output "users" {
  value = {
    alice   = arize_user.data_scientist.id
    bob     = arize_user.ml_engineer.id
    charlie = arize_user.analyst.id
  }
  description = "User IDs"
}

# ============================================================================
# WORKFLOW: How SAML SSO Works
# ============================================================================
#
# 1. Configure SAML IdP with metadata URL or XML
#    - Set email domains that are allowed
#    - Set default role for unmapped users
#    - Enable sync_user_roles to map SAML attributes to Arize roles
#
# 2. First SAML login:
#    - User logs in through your identity provider (Okta, Azure AD, etc)
#    - Arize receives SAML assertion with user attributes
#    - User is auto-created in Arize with default role
#    - User can be assigned to spaces with terraform apply
#
# 3. Role mapping:
#    - For simple setups: use default_org_role_id + space assignments
#    - For complex setups: use SAML attributes mapping (requires API extensions)
#
# 4. Subsequent logins:
#    - User logs in via SAML and gets provisioned automatically
#    - Space membership is managed by Terraform
#
# ============================================================================
# ADVANTAGES OF THIS SETUP
# ============================================================================
#
# ✅ Centralized identity management via SAML/SSO
# ✅ Automatic user provisioning on first login
# ✅ Terraform-managed team structure and permissions
# ✅ Audit trail through terraform state
# ✅ Easy offboarding (remove user from spaces, disable in IdP)
# ✅ Consistent role assignment across teams
#
# ============================================================================
# COMMON WORKFLOWS
# ============================================================================
#
# Onboard a new team member:
#   1. Add them to your identity provider (Okta, Azure, etc)
#   2. Create arize_user resource for them
#   3. Create arize_space_member resources for their teams
#   4. terraform apply
#   5. They can now log in via SAML
#
# Offboard a team member:
#   1. Remove them from your identity provider
#   2. Remove or comment out their arize_user resource
#   3. Remove arize_space_member resources
#   4. terraform apply
#   5. They no longer have access to Arize
#
# Add new team/space:
#   1. Create new arize_space resource
#   2. Create arize_space_member resources for team members
#   3. terraform apply
#
# Rotate team leads:
#   1. Change role from "admin" to "member" for old lead
#   2. Change role from "member" to "admin" for new lead
#   3. terraform apply
