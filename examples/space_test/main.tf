terraform {
  required_providers {
    arize = {
      source = "arize-ai/arize"
    }
  }
}

provider "arize" {}

# Create a new space
resource "arize_space" "test" {
  name    = "tf-test-space-$(date +%s)"
  private = false
  description = "Test space created via Terraform"
}

# Add Sean Lee to the space as a member
resource "arize_space_member" "sean" {
  space_id = arize_space.test.id
  user_id  = "VXNlcjoxMjczOTpaemlJ"  # Sean Lee (yihsean@gmail.com)
  role     = "member"
}

output "space" {
  value = arize_space.test
}

output "space_member" {
  value = arize_space_member.sean
}
