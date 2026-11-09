# Example 5: Importing Existing Resources & Using Data Sources
#
# Manage Arize resources that were created outside Terraform: look them up
# with data sources, import the ones Terraform should own, and connect them
# to new resources.

terraform {
  required_providers {
    arize = {
      source  = "seanlee10/arize"
      version = "~> 0.3.6"
    }
  }
}

provider "arize" {}

variable "company_domain" {
  description = "Email domain of your company's users"
  type        = string
  default     = "company.com"
}

variable "space_admin_email" {
  description = "Email of an existing user to make admin of the new space"
  type        = string
  default     = "alice@company.com"
}

# ============================================================================
# STEP 1: List existing resources with data sources
# ============================================================================

data "arize_users" "all" {}

data "arize_roles" "all" {}

locals {
  company_users = [
    for user in data.arize_users.all.users :
    user if endswith(lower(user.email), "@${lower(var.company_domain)}")
  ]

  active_company_users = [for user in local.company_users : user if user.status == "active"]
}

output "user_analysis" {
  value = {
    total_users_in_account = length(data.arize_users.all.users)
    company_users          = length(local.company_users)
    active_company_users   = length(local.active_company_users)
    company_users_list = [
      for user in local.company_users : {
        id     = user.id
        email  = user.email
        name   = user.name
        status = user.status
      }
    ]
  }
  description = "Existing users in the Arize account"
}

output "existing_roles" {
  value = {
    for role in data.arize_roles.all.roles : role.name => {
      id            = role.id
      is_predefined = role.is_predefined
    }
  }
  description = "All roles in the account"
}

# ============================================================================
# STEP 2: Import existing resources into Terraform
# ============================================================================
#
# Write a resource block that matches the existing resource, import it, then
# run `terraform plan`; it should show no changes (adjust the config if it does).
#
#   terraform import arize_user.imported_user "<user_id or pending invitation_id>"
#   terraform import arize_space.imported_space "<space_id>"
#   terraform import arize_space_member.imported_member "<space_id>:<user_id>"
#   terraform import arize_role.imported_role "<role_id>"
#   terraform import arize_evaluator.imported_eval "<space_id>/<evaluator_id>"
#   terraform import arize_dataset.imported_dataset "<dataset_id>:<key_column>"
#   terraform import arize_saml_idp.imported_idp "<idp_id>"
#
# On Terraform 1.5+ you can use import blocks instead:
#
#   import {
#     to = arize_space.imported_space
#     id = "<space_id>"
#   }

# After: terraform import arize_user.imported_user "<user_id>"
resource "arize_user" "imported_user" {
  email = "existing@company.com"
  name  = "Existing User"
}

# After: terraform import arize_space.imported_space "<space_id>"
resource "arize_space" "imported_space" {
  name        = "existing-space"
  private     = false
  description = "Space created in the Arize UI"
}

# After: terraform import arize_space_member.imported_member "<space_id>:<user_id>"
# (Import the user first: a newly invited user can't join a space until they
# accept the invitation.)
resource "arize_space_member" "imported_member" {
  space_id = arize_space.imported_space.id
  user_id  = arize_user.imported_user.id
  role     = "member"
}

# ============================================================================
# STEP 3: Connect existing users to new resources
# ============================================================================

locals {
  active_user_ids = {
    for user in data.arize_users.all.users : lower(user.email) => user.id
    if user.status == "active"
  }
  space_admin_id = lookup(local.active_user_ids, lower(var.space_admin_email), null)
}

resource "arize_space" "new_collab_space" {
  name        = "collaboration-with-existing-users"
  private     = false
  description = "New space with existing team members"
}

# Added only if the user exists and has accepted their invitation
resource "arize_space_member" "existing_user_in_new_space" {
  count    = local.space_admin_id != null ? 1 : 0
  space_id = arize_space.new_collab_space.id
  user_id  = local.space_admin_id
  role     = "admin"
}
