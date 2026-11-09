# Example 3: Datasets Synced from Files (CI/CD)
#
# Keeps an Arize dataset in sync with a CSV file in your repository. Run
# `terraform apply` from CI on every merge, and edits to the CSV become edits
# to the dataset:
#   - a changed row updates its example in place
#   - a new row adds an example; a removed row deletes its example
#   - every apply edits the dataset's latest version (no new version)
#
#   terraform apply -var="space_id=<space_id>"

terraform {
  required_providers {
    arize = {
      source  = "seanlee10/arize"
      version = "~> 0.3.6"
    }
  }
}

# In CI, set ARIZE_API_KEY from your secret store.
provider "arize" {}

variable "space_id" {
  description = "Space that owns the dataset"
  type        = string
}

# Rows are matched to existing examples by key_column, so its values must be
# unique. Each row must include the key column with the same value as its key.
resource "arize_dataset" "basic_quiz" {
  space_id   = var.space_id
  name       = "basic-quiz"
  key_column = "question"
  examples = {
    for row in csvdecode(file("${path.module}/datasets/basic_quiz.csv")) : row.question => row
  }
}

# The same works for JSON: a list of objects with string values.
# resource "arize_dataset" "from_json" {
#   space_id   = var.space_id
#   name       = "from-json"
#   key_column = "question"
#   examples   = { for row in jsondecode(file("${path.module}/datasets/rows.json")) : row.question => row }
# }

output "dataset_id" {
  value       = arize_dataset.basic_quiz.id
  description = "ID of the dataset (import with \"<dataset_id>:question\")"
}

# Example GitHub Actions step:
#
#   - run: terraform init && terraform apply -auto-approve -var="space_id=${{ vars.ARIZE_SPACE_ID }}"
#     env:
#       ARIZE_API_KEY: ${{ secrets.ARIZE_API_KEY }}
