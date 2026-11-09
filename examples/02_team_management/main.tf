# Example 2: Team Management - Inviting a Team and Assigning Space Roles
#
# Invites every team member and adds them to spaces with their role.
#
# Arize can add a user to a space only after they accept their invitation.
# So the space memberships below cover only team members who are already
# active users: the first apply sends the invitations, and once someone has
# accepted, the next `terraform apply` adds them to the spaces.

terraform {
  required_providers {
    arize = {
      source  = "seanlee10/arize"
      version = "~> 0.3.6"
    }
  }
}

provider "arize" {}

# Define team members (key => email, name, space role)
locals {
  team = {
    alice = {
      email = "alice@company.com"
      name  = "Alice Johnson"
      role  = "admin" # Can manage the space and its members
    }
    bob = {
      email = "bob@company.com"
      name  = "Bob Smith"
      role  = "member" # Can view and edit content
    }
    carol = {
      email = "carol@company.com"
      name  = "Carol White"
      role  = "readOnly" # Can view but not edit
    }
    david = {
      email = "david@company.com"
      name  = "David Brown"
      role  = "annotator" # Can annotate and give feedback
    }
  }
}

# Invite all team members. Until a user accepts, their status is "invited".
# Arize refuses to invite someone who already has an account; import those
# users instead: terraform import 'arize_user.team_members["bob"]' "<user_id>"
resource "arize_user" "team_members" {
  for_each = local.team

  email = each.value.email
  name  = each.value.name
}

# Users who have accepted their invitation, by lowercased email
data "arize_users" "all" {}

locals {
  active_user_ids = {
    for user in data.arize_users.all.users : lower(user.email) => user.id
    if user.status == "active"
  }

  # Team members who can be added to spaces now
  active_team = {
    for key, member in local.team : key => merge(member, {
      user_id = local.active_user_ids[lower(member.email)]
    })
    if contains(keys(local.active_user_ids), lower(member.email))
  }
}

resource "arize_space" "ml_team" {
  name        = "ml-team-models"
  private     = false
  description = "Collaborative space for the ML team"
}

# Add each active team member to the space with their role
resource "arize_space_member" "team_access" {
  for_each = local.active_team

  space_id = arize_space.ml_team.id
  user_id  = each.value.user_id
  role     = each.value.role
}

# A separate space for read-only stakeholders
resource "arize_space" "stakeholder_view" {
  name        = "stakeholder-dashboard"
  private     = false
  description = "Read-only space for business stakeholders"
}

resource "arize_space_member" "stakeholder_access" {
  for_each = {
    for key, member in local.active_team : key => member
    if member.role == "readOnly"
  }

  space_id = arize_space.stakeholder_view.id
  user_id  = each.value.user_id
  role     = "readOnly"
}

output "team_summary" {
  value = {
    for key, member in local.team : key => {
      email  = member.email
      role   = member.role
      status = arize_user.team_members[key].status
      spaces = contains(keys(local.active_team), key) ? compact([
        arize_space.ml_team.name,
        member.role == "readOnly" ? arize_space.stakeholder_view.name : "",
      ]) : []
    }
  }
  description = "Team members, their roles, invitation status, and assigned spaces"
}

output "pending_invitations" {
  value       = [for key, member in local.team : member.email if !contains(keys(local.active_team), key)]
  description = "Team members who haven't accepted their invitation yet (rerun apply after they do)"
}

output "ml_team_space_id" {
  value       = arize_space.ml_team.id
  description = "Main ML team collaboration space"
}

output "stakeholder_space_id" {
  value       = arize_space.stakeholder_view.id
  description = "Stakeholder read-only dashboard space"
}
