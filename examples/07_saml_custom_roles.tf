# Example 7: SAML SSO with Custom Roles
# This example demonstrates using custom roles with SAML IdP configuration

terraform {
  required_providers {
    arize = {
      source = "arize-ai/arize"
    }
  }
}

provider "arize" {}

# ============================================================================
# STEP 1: Create Custom Roles
# ============================================================================

# Custom role for data scientists with specific permissions
resource "arize_role" "data_scientist" {
  name        = "data-scientist"
  description = "Data scientist role with model monitoring permissions"
  # permissions = [...]  # Define specific permissions as needed
}

# Custom role for ML operations team
resource "arize_role" "ml_ops_engineer" {
  name        = "ml-ops-engineer"
  description = "ML operations engineer with deployment and monitoring access"
}

# Custom role for data analysts - read-only access
resource "arize_role" "analyst" {
  name        = "analyst"
  description = "Analyst role with read-only access to dashboards"
}

# ============================================================================
# STEP 2: Configure SAML IdP with Custom Default Role
# ============================================================================

resource "arize_saml_idp" "company_saml" {
  metadata_url = "https://idp.company.com/metadata.xml"

  email_domains_list = ["company.com"]

  # Use custom role as default for all SAML users
  # This role ID comes from the arize_role resource above
  default_org_role_id = arize_role.data_scientist.id

  enforce_saml        = false
  sync_user_roles     = true
  allow_login_with_defaults = true
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

# ============================================================================
# STEP 4: Create Users with Different Custom Roles
# ============================================================================

# Data scientist user - gets custom data-scientist role
resource "arize_user" "alice_ds" {
  email = "alice@company.com"
  name  = "Alice - Data Scientist"
}

# ML ops engineer - gets custom ml-ops-engineer role
resource "arize_user" "bob_mlops" {
  email = "bob@company.com"
  name  = "Bob - ML Ops Engineer"
}

# Analyst - gets custom analyst role
resource "arize_user" "charlie_analyst" {
  email = "charlie@company.com"
  name  = "Charlie - Data Analyst"
}

# ============================================================================
# STEP 5: Assign Users to Spaces with Custom Roles
# ============================================================================

# Data scientist in data science space (admin)
resource "arize_space_member" "alice_ds_space" {
  space_id = arize_space.data_science_team.id
  user_id  = arize_user.alice_ds.id
  role     = "admin"  # Space role (different from org role)
}

# ML ops engineer in ml ops space (admin)
resource "arize_space_member" "bob_mlops_space" {
  space_id = arize_space.ml_ops_team.id
  user_id  = arize_user.bob_mlops.id
  role     = "admin"
}

# Analyst in analytics space (read-only)
resource "arize_space_member" "charlie_analyst_space" {
  space_id = arize_space.analytics_team.id
  user_id  = arize_user.charlie_analyst.id
  role     = "readOnly"
}

# ============================================================================
# STEP 6: Create API Keys with Custom Roles
# ============================================================================

# Service account for automated monitoring using custom role permissions
resource "arize_api_key" "monitoring_service" {
  name = "monitoring-automation"
  # Permissions tied to the ml-ops-engineer custom role
  permissions = ["read:models", "write:predictions"]
}

# ============================================================================
# OUTPUTS
# ============================================================================

output "custom_roles" {
  value = {
    data_scientist   = arize_role.data_scientist.id
    ml_ops_engineer  = arize_role.ml_ops_engineer.id
    analyst          = arize_role.analyst.id
  }
  description = "Custom role IDs for reference"
}

output "saml_idp_with_custom_role" {
  value       = arize_saml_idp.company_saml.id
  description = "SAML IdP configured with custom data_scientist role"
}

output "team_structure" {
  value = {
    data_science = {
      space_id = arize_space.data_science_team.id
      users = [
        {
          name     = arize_user.alice_ds.name
          email    = arize_user.alice_ds.email
          role     = "admin"
        }
      ]
    }
    ml_ops = {
      space_id = arize_space.ml_ops_team.id
      users = [
        {
          name     = arize_user.bob_mlops.name
          email    = arize_user.bob_mlops.email
          role     = "admin"
        }
      ]
    }
    analytics = {
      space_id = arize_space.analytics_team.id
      users = [
        {
          name     = arize_user.charlie_analyst.name
          email    = arize_user.charlie_analyst.email
          role     = "readOnly"
        }
      ]
    }
  }
  description = "Team structure with custom roles"
}

# ============================================================================
# USAGE NOTES
# ============================================================================
#
# 1. ORGANIZATION ROLES vs SPACE ROLES:
#    - Organization roles (default_org_role_id): Control Arize account permissions
#    - Space roles (role in arize_space_member): Control space-level access
#
# 2. USING CUSTOM ROLES:
#    - Create custom roles with arize_role resource
#    - Reference them by ID in default_org_role_id
#    - Each SAML user gets the default custom role on first login
#    - Assign to spaces with specific space roles for granular control
#
# 3. PERMISSION INHERITANCE:
#    - Custom org role → grants account-level permissions
#    - Space role → grants space-level access
#    - Combine both for fine-grained control
#
# 4. WORKFLOW:
#    - User logs in via SAML
#    - Automatically assigned custom org role (from default_org_role_id)
#    - Can be assigned to spaces with different space roles
#    - Space role takes precedence in that space
#
# ============================================================================
# EXAMPLE: DIFFERENT CUSTOM ROLES FOR DIFFERENT TEAMS
# ============================================================================
#
# To assign different custom roles based on SAML attributes:
# 1. Create multiple custom roles
# 2. Create multiple SAML IdP configurations (one per team)
# 3. Each SAML IdP has its own default_org_role_id pointing to custom role
# 4. Use email_domains_list to scope each IdP to a team
#
# This allows fine-grained control over who gets which custom role!
