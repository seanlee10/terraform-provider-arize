# Example 2: Team Management - Multiple Users and Roles
# This example demonstrates managing a team with different roles and responsibilities

terraform {
  required_providers {
    arize = {
      source = "arize-ai/arize"
    }
  }
}

provider "arize" {}

# Define team members
locals {
  team = {
    alice = {
      email = "alice@company.com"
      name  = "Alice Johnson"
      role  = "admin"  # Can manage space
    }
    bob = {
      email = "bob@company.com"
      name  = "Bob Smith"
      role  = "member"  # Can edit models
    }
    carol = {
      email = "carol@company.com"
      name  = "Carol White"
      role  = "readOnly"  # Can view but not edit
    }
    david = {
      email = "david@company.com"
      name  = "David Brown"
      role  = "annotator"  # Can provide feedback
    }
  }
}

# Create all team members
resource "arize_user" "team_members" {
  for_each = local.team

  email = each.value.email
  name  = each.value.name
}

# Create team collaboration space
resource "arize_space" "ml_team" {
  name        = "ml-team-models"
  private     = false
  description = "Collaborative space for ML team to monitor and evaluate models"
}

# Add each team member to the space with their assigned role
resource "arize_space_member" "team_access" {
  for_each = local.team

  space_id = arize_space.ml_team.id
  user_id  = arize_user.team_members[each.key].id
  role     = each.value.role
}

# Create separate space for read-only stakeholders
resource "arize_space" "stakeholder_view" {
  name        = "stakeholder-dashboard"
  private     = false
  description = "Read-only space for business stakeholders to view model metrics"
}

# Add read-only stakeholder access
resource "arize_space_member" "stakeholder_access" {
  for_each = {
    for name, member in local.team :
    name => member if member.role == "readOnly"
  }

  space_id = arize_space.stakeholder_view.id
  user_id  = arize_user.team_members[each.key].id
  role     = "readOnly"
}

# Outputs for reference
output "team_summary" {
  value = {
    for name, member in local.team : name => {
      user_id = arize_user.team_members[name].id
      email   = member.email
      role    = member.role
      spaces = [
        arize_space.ml_team.name,
        member.role == "readOnly" ? arize_space.stakeholder_view.name : null
      ]
    }
  }
  description = "Summary of team members, their emails, roles, and assigned spaces"
}

output "ml_team_space_id" {
  value = arize_space.ml_team.id
  description = "Main ML team collaboration space"
}

output "stakeholder_space_id" {
  value = arize_space.stakeholder_view.id
  description = "Stakeholder read-only dashboard space"
}
