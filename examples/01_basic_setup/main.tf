# Example 1: Basic Setup - A Space and an Existing User
#
# Creates a space and adds a user who already has an Arize account.
# Arize can add users to spaces only after they accept their invitation, so
# this example looks the user up instead of inviting them (see example 2 for
# inviting new users).
#
#   terraform apply -var="user_email=alice@company.com"

terraform {
  required_providers {
    arize = {
      source  = "seanlee10/arize"
      version = "~> 0.3.6"
    }
  }
}

provider "arize" {
  # API key loaded from the ARIZE_API_KEY environment variable
}

variable "user_email" {
  description = "Email of an existing (active) Arize user to add to the space"
  type        = string
}

# Users who have already accepted their invitation
data "arize_users" "all" {}

locals {
  user_ids = {
    for user in data.arize_users.all.users : lower(user.email) => user.id
    if user.status == "active"
  }
  user_id = lookup(local.user_ids, lower(var.user_email), null)
}

resource "arize_space" "example_space" {
  name        = "example-space"
  private     = false
  description = "Example space for getting started"
}

resource "arize_space_member" "user_in_space" {
  space_id = arize_space.example_space.id
  user_id  = local.user_id
  role     = "member"

  lifecycle {
    precondition {
      condition     = local.user_id != null
      error_message = "No active Arize user has that email. Invite them first (arize_user) and wait until they accept."
    }
  }
}

output "space_id" {
  value       = arize_space.example_space.id
  description = "ID of the created space"
}

output "space_uuid" {
  value       = arize_space.example_space.uuid
  description = "UUID of the space"
}

output "user_id" {
  value       = local.user_id
  description = "ID of the user added to the space"
}
