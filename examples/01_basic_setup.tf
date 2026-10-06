# Example 1: Basic Setup - Single User and Space
# This example shows the simplest way to get started with the Arize provider

terraform {
  required_providers {
    arize = {
      source = "arize-ai/arize"
    }
  }
}

provider "arize" {
  # API key loaded from ARIZE_API_KEY environment variable
}

# Create a simple user
resource "arize_user" "example_user" {
  email = "user@example.com"
  name  = "Example User"
}

# Create a simple space
resource "arize_space" "example_space" {
  name    = "example-space"
  private = false
  description = "Example space for getting started"
}

# Add the user to the space
resource "arize_space_member" "user_in_space" {
  space_id = arize_space.example_space.id
  user_id  = arize_user.example_user.id
  role     = "member"
}

# Output the important IDs
output "user_id" {
  value = arize_user.example_user.id
  description = "ID of the created user"
}

output "space_id" {
  value = arize_space.example_space.id
  description = "ID of the created space"
}

output "space_uuid" {
  value = arize_space.example_space.uuid
  description = "UUID of the space (use for API calls)"
}
