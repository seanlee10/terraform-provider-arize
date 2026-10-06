terraform {
  required_providers {
    arize = {
      source = "arize-ai/arize"
    }
  }
}

provider "arize" {}

resource "arize_user" "sean" {
  email = "yihsean@gmail.com"
}

output "user" {
  value = arize_user.sean
}
