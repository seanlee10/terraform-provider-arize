# Example 7: SAML SSO with Custom Roles
# This example demonstrates using custom roles with SAML IdP configuration

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
# STEP 1: Create Custom Roles
# ============================================================================

# Custom role for data scientists with specific permissions
resource "arize_role" "data_scientist" {
  name        = "data-scientist"
  description = "Data scientist role with project, dataset, and experiment access"
  permissions = [
    "PROJECT_READ",
    "PROJECT_SPAN_READ",
    "PROJECT_SPAN_ANNOTATE",
    "DATASET_READ",
    "DATASET_CREATE",
    "EXPERIMENT_READ",
    "EXPERIMENT_CREATE",
    "EVALUATOR_READ",
  ]
}

# Custom role for ML operations team
resource "arize_role" "ml_ops_engineer" {
  name        = "ml-ops-engineer"
  description = "ML operations engineer with monitoring and evaluator access"
  permissions = [
    "PROJECT_READ",
    "MONITOR_READ",
    "MONITOR_CREATE",
    "MONITOR_UPDATE",
    "EVALUATOR_READ",
    "EVALUATOR_CREATE",
    "EVALUATOR_UPDATE",
  ]
}

# Custom role for data analysts - read-only access
resource "arize_role" "analyst" {
  name        = "analyst"
  description = "Analyst role with read-only access to dashboards"
  permissions = [
    "DASHBOARD_READ",
    "PROJECT_READ",
  ]
}

# ============================================================================
# STEP 2: Map IdP Groups to Custom Roles
# ============================================================================

# Custom roles apply per space (space_custom_roles) or per project. The
# organization role (org_role) and the default_* roles must be built-in
# names: admin, member, readOnly, annotator.
resource "arize_saml_idp" "company_saml" {
  metadata_url = "https://idp.company.com/metadata.xml"

  email_domains_list = ["company.com"]

  enforce_saml    = false
  sync_user_roles = true

  role_mappings = [
    {
      attributes         = { groups = "data-science" }
      org_id             = var.org_id
      org_role           = "member"
      space_custom_roles = { (arize_space.data_science_team.id) = arize_role.data_scientist.id }
    },
    {
      attributes         = { groups = "ml-ops" }
      org_id             = var.org_id
      org_role           = "member"
      space_custom_roles = { (arize_space.ml_ops_team.id) = arize_role.ml_ops_engineer.id }
    },
    {
      attributes         = { groups = "analytics" }
      org_id             = var.org_id
      org_role           = "readOnly"
      space_custom_roles = { (arize_space.analytics_team.id) = arize_role.analyst.id }
    },
  ]
}

# ============================================================================
# STEP 3: Create Spaces
# ============================================================================

resource "arize_space" "data_science_team" {
  name        = "data-science-team"
  private     = true
  description = "Data science team space"
}

resource "arize_space" "ml_ops_team" {
  name        = "ml-ops-team"
  private     = true
  description = "ML operations team space"
}

resource "arize_space" "analytics_team" {
  name        = "analytics-team"
  private     = false
  description = "Analytics and insights space"
}

# Users get these roles on their SAML login; no arize_user or
# arize_space_member resources are needed. arize_space_member takes only the
# built-in space roles (admin, member, readOnly, annotator).

# ============================================================================
# OUTPUTS
# ============================================================================

output "custom_roles" {
  value = {
    data_scientist  = arize_role.data_scientist.id
    ml_ops_engineer = arize_role.ml_ops_engineer.id
    analyst         = arize_role.analyst.id
  }
  description = "Custom role IDs for reference"
}

output "saml_idp_with_custom_role" {
  value       = arize_saml_idp.company_saml.id
  description = "SAML IdP mapping IdP groups to custom roles"
}

output "team_structure" {
  value = {
    data-science = {
      space_id = arize_space.data_science_team.id
      org_role = "member"
      role_id  = arize_role.data_scientist.id
    }
    ml-ops = {
      space_id = arize_space.ml_ops_team.id
      org_role = "member"
      role_id  = arize_role.ml_ops_engineer.id
    }
    analytics = {
      space_id = arize_space.analytics_team.id
      org_role = "readOnly"
      role_id  = arize_role.analyst.id
    }
  }
  description = "Custom space role each IdP group gets"
}

# ============================================================================
# USAGE NOTES
# ============================================================================
#
# 1. ORGANIZATION ROLES vs SPACE ROLES:
#    - Organization roles (org_role, default_org_role_id): built-in names only
#    - Space roles: built-in (space_roles) or custom (space_custom_roles)
#    - Project roles (project_roles): custom or predefined role IDs
#
# 2. USING CUSTOM ROLES:
#    - Create custom roles with the arize_role resource
#    - Reference them by ID in space_custom_roles / project_roles
#    - Arize rejects custom role IDs in org_role and default_*_role_id
#
# 3. WORKFLOW:
#    - User logs in via SAML
#    - Arize matches the user's SAML attributes against role_mappings
#    - The matching mapping sets their org role and custom space roles
#    - With sync_user_roles = true, this is re-applied on every login
#
# ============================================================================
# EXAMPLE: DIFFERENT CUSTOM ROLES FOR DIFFERENT TEAMS
# ============================================================================
#
# The role_mappings above already give each IdP group its own custom role.
# A mapping matches when the user has every attribute in `attributes`, so
# combine attributes to narrow it:
#
#   attributes = { groups = "data-science", department = "research" }
