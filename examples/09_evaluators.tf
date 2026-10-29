# Example 9: Evaluators
# Manage LLM-as-judge (template) and Python (code) evaluators as code.
#
# Arize versions evaluators. Changing `template` or `code` publishes a new
# evaluator version with `commit_message`; `name` and `description` are edited
# in place. Changing `space_id` or `data_granularity`, or switching between
# `template` and `code`, replaces the evaluator.

terraform {
  required_providers {
    arize = {
      source = "seanlee10/arize"
    }
  }
}

provider "arize" {}

variable "space_id" {
  description = "Space that owns the evaluators"
  type        = string
}

variable "llm_integration_id" {
  description = "LLM integration used as the judge (Settings -> AI Providers in Arize)"
  type        = string
}

# ============================================================================
# LLM-as-judge evaluator
# ============================================================================

resource "arize_evaluator" "correctness" {
  space_id       = var.space_id
  name           = "answer-correctness"
  description    = "Judges whether the answer is correct for the question"
  commit_message = "Initial correctness judge"

  template = {
    # Result column written on each span; must be unique in the space
    column_name = "correctness"

    template = <<-EOT
      You are checking whether an answer is correct.
      [BEGIN DATA]
      Question: {input}
      Answer: {output}
      [END DATA]
      Respond with exactly one word: "correct" or "incorrect".
    EOT

    # Label -> score
    classification_choices = {
      correct   = 1
      incorrect = 0
    }
    direction            = "maximize"
    include_explanations = true

    # Optional: omit to use the space's default LLM integration
    llm = {
      integration_id        = var.llm_integration_id
      model_name            = "gpt-4o"
      invocation_parameters = jsonencode({ temperature = 0 })
    }
  }
}

# Score whole traces instead of individual spans
resource "arize_evaluator" "user_frustration" {
  space_id         = var.space_id
  name             = "user-frustration"
  data_granularity = "trace"

  template = {
    column_name = "user_frustration"
    template    = <<-EOT
      Here is a conversation between a user and an assistant:
      {input}
      Did the user become frustrated? Respond with "frustrated" or "ok".
    EOT
    classification_choices = {
      frustrated = 1
      ok         = 0
    }
    direction = "minimize"
  }
}

# ============================================================================
# Python code evaluator
# ============================================================================

resource "arize_evaluator" "non_empty_output" {
  space_id = var.space_id
  name     = "non-empty-output"

  code = {
    column_name = "non_empty_output"
    class_name  = "NonEmptyOutput"

    imports = <<-EOT
      from typing import Any, Optional
      from arize.experimental.datasets.experiments.evaluators.base import (
          EvaluationResult,
          CodeEvaluator,
      )
    EOT

    # Arize derives span_attributes from the evaluate() keyword arguments
    code = <<-EOT
      class NonEmptyOutput(CodeEvaluator):
          def evaluate(self, *, output: Optional[str] = None, **kwargs: Any) -> EvaluationResult:
              ok = bool(output and output.strip())
              return EvaluationResult(score=1.0 if ok else 0.0, label="non_empty" if ok else "empty")
    EOT
  }
}

# ============================================================================
# Evaluators created in the Arize UI
# ============================================================================

data "arize_evaluators" "all" {
  space_id = var.space_id
}

locals {
  evaluator_ids = { for e in data.arize_evaluators.all.evaluators : e.name => e.id }
}

output "evaluators" {
  value       = local.evaluator_ids
  description = "All evaluators in the space, by name"
}

output "correctness_version" {
  value = arize_evaluator.correctness.version_number
}

output "non_empty_span_attributes" {
  value = arize_evaluator.non_empty_output.code.span_attributes
}

# ============================================================================
# IMPORTING AN EXISTING EVALUATOR
# ============================================================================
#
# Import IDs are "<space_id>/<evaluator_id>". Find evaluator IDs with the
# arize_evaluators data source above, then:
#
#   terraform import arize_evaluator.my_eval "<space_id>/<evaluator_id>"
#
# After importing a template evaluator, set llm.invocation_parameters /
# provider_parameters to match what you want: Arize normalizes those per
# provider, so they are not read back into state.
