package client

import (
	"context"
	"fmt"
	"strings"
)

type Evaluator struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Description   *string          `json:"description"`
	TaskType      string           `json:"taskType"`
	CommitMessage string           `json:"commitMessage"`
	CreatedAt     string           `json:"createdAt"`
	UpdatedAt     string           `json:"updatedAt"`
	Config        *EvaluatorConfig `json:"config"`
}

// EvaluatorConfig flattens the TemplateEvaluationConfig | CodeEvaluationConfig union.
type EvaluatorConfig struct {
	Typename            string `json:"__typename"`
	Name                string `json:"name"`
	DataGranularityType string `json:"dataGranularityType"`
	EvaluatorVersion    *struct {
		VersionNumber int64 `json:"versionNumber"`
	} `json:"evaluatorVersion"`

	Template                       string             `json:"template"`
	ClassificationChoices          map[string]float64 `json:"classificationChoices"`
	Direction                      string             `json:"direction"`
	IncludeExplanations            bool               `json:"includeExplanations"`
	UseFunctionCallingIfAvailable  bool               `json:"useFunctionCallingIfAvailable"`
	UseStructuredOutputIfAvailable bool               `json:"useStructuredOutputIfAvailable"`
	LlmConfig                      *struct {
		IntegrationID string `json:"integrationId"`
		ModelName     string `json:"modelName"`
	} `json:"llmConfig"`

	EvaluationClass    string   `json:"evaluationClass"`
	EvalClassCodeBlock string   `json:"evalClassCodeBlock"`
	PackageImports     string   `json:"packageImports"`
	SpanAttributes     []string `json:"spanAttributes"`
}

const (
	EvaluatorConfigTemplate = "TemplateEvaluationConfig"
	EvaluatorConfigCode     = "CodeEvaluationConfig"
)

type TemplateEvaluatorInput struct {
	Name                           string             `json:"name"`
	Template                       string             `json:"template"`
	Position                       int                `json:"position"`
	ClassificationChoices          map[string]float64 `json:"classificationChoices,omitempty"`
	Direction                      string             `json:"direction,omitempty"`
	IncludeExplanations            bool               `json:"includeExplanations"`
	UseFunctionCallingIfAvailable  bool               `json:"useFunctionCallingIfAvailable"`
	UseStructuredOutputIfAvailable bool               `json:"useStructuredOutputIfAvailable"`
	DataGranularityType            string             `json:"dataGranularityType"`
	LlmConfig                      *LlmConfigInput    `json:"llmConfig,omitempty"`
}

// All four fields are NON_NULL in the API schema.
type LlmConfigInput struct {
	IntegrationID        string                 `json:"integrationId"`
	ModelName            string                 `json:"modelName"`
	InvocationParameters map[string]interface{} `json:"invocationParameters"`
	ProviderParameters   map[string]interface{} `json:"providerParameters"`
}

type CodeEvaluatorInput struct {
	Name                string `json:"name"`
	EvaluationClass     string `json:"evaluationClass"`
	EvalClassCodeBlock  string `json:"evalClassCodeBlock"`
	PackageImports      string `json:"packageImports,omitempty"`
	Position            int    `json:"position"`
	DataGranularityType string `json:"dataGranularityType"`
}

type EvaluatorConfigInput struct {
	Template *TemplateEvaluatorInput
	Code     *CodeEvaluatorInput
}

func (c EvaluatorConfigInput) apply(input map[string]interface{}) {
	if c.Template != nil {
		input["templateEvaluator"] = c.Template
	}
	if c.Code != nil {
		input["codeEvaluator"] = c.Code
	}
}

const evaluatorFields = `
  id
  name
  description
  taskType
  commitMessage
  createdAt
  updatedAt
  config {
    __typename
    ... on TemplateEvaluationConfig {
      name
      dataGranularityType
      evaluatorVersion { versionNumber }
      template
      classificationChoices
      direction
      includeExplanations
      useFunctionCallingIfAvailable
      useStructuredOutputIfAvailable
      llmConfig { integrationId modelName }
    }
    ... on CodeEvaluationConfig {
      name
      dataGranularityType
      evaluatorVersion { versionNumber }
      evaluationClass
      evalClassCodeBlock
      packageImports
      spanAttributes
    }
  }
`

// ErrEvaluatorNotFound is returned when the evaluator no longer exists in its space.
var ErrEvaluatorNotFound = fmt.Errorf("evaluator not found")

func (c *Client) GetEvaluator(ctx context.Context, spaceID, evaluatorID string) (*Evaluator, error) {
	var result struct {
		Node *Evaluator `json:"node"`
	}
	query := `query GetEvaluator($id: ID!) { node(id: $id) { ... on Evaluator {` + evaluatorFields + `} } }`

	err := c.execute(ctx, query, map[string]interface{}{"id": evaluatorID}, &result)
	if err == nil && result.Node != nil && result.Node.ID != "" {
		return result.Node, nil
	}

	// Arize answers node(id) for a deleted evaluator with "Unauthorized access"
	// rather than null, so confirm absence via the space listing before reporting it gone.
	if err != nil && !strings.Contains(err.Error(), "Unauthorized") {
		return nil, fmt.Errorf("failed to get evaluator: %w", err)
	}
	evaluators, listErr := c.ListEvaluators(ctx, spaceID)
	if listErr != nil {
		return nil, fmt.Errorf("failed to get evaluator: %w", listErr)
	}
	for _, e := range evaluators {
		if e.ID == evaluatorID {
			if err != nil {
				return nil, fmt.Errorf("failed to get evaluator: %w", err)
			}
			return nil, fmt.Errorf("failed to get evaluator %s: empty response", evaluatorID)
		}
	}
	return nil, ErrEvaluatorNotFound
}

func (c *Client) ListEvaluators(ctx context.Context, spaceID string) ([]*Evaluator, error) {
	const query = `
query ListEvaluators($spaceId: ID!, $first: Int!, $after: String) {
  node(id: $spaceId) {
    ... on Space {
      evaluators(first: $first, after: $after) {
        pageInfo { hasNextPage endCursor }
        edges { node { id name description taskType createdAt updatedAt } }
      }
    }
  }
}`
	var evaluators []*Evaluator
	var after interface{}
	for {
		var result struct {
			Node *struct {
				Evaluators struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Edges []struct {
						Node *Evaluator `json:"node"`
					} `json:"edges"`
				} `json:"evaluators"`
			} `json:"node"`
		}
		vars := map[string]interface{}{"spaceId": spaceID, "first": 50, "after": after}
		if err := c.execute(ctx, query, vars, &result); err != nil {
			return nil, fmt.Errorf("failed to list evaluators: %w", err)
		}
		if result.Node == nil {
			return nil, fmt.Errorf("space %s not found", spaceID)
		}
		for _, edge := range result.Node.Evaluators.Edges {
			if edge.Node != nil {
				evaluators = append(evaluators, edge.Node)
			}
		}
		if !result.Node.Evaluators.PageInfo.HasNextPage {
			return evaluators, nil
		}
		after = result.Node.Evaluators.PageInfo.EndCursor
	}
}

func (c *Client) CreateEvaluator(ctx context.Context, spaceID, name, description, commitMessage string, config EvaluatorConfigInput) (string, error) {
	const query = `
mutation CreateEvaluator($input: CreateEvaluatorMutationInput!) {
  createEvaluator(input: $input) { evaluator { id } }
}`
	input := map[string]interface{}{
		"spaceId":       spaceID,
		"name":          name,
		"commitMessage": commitMessage,
	}
	if description != "" {
		input["description"] = description
	}
	config.apply(input)

	var result struct {
		CreateEvaluator struct {
			Evaluator *struct {
				ID string `json:"id"`
			} `json:"evaluator"`
		} `json:"createEvaluator"`
	}
	if err := c.execute(ctx, query, map[string]interface{}{"input": input}, &result); err != nil {
		return "", fmt.Errorf("failed to create evaluator: %w", err)
	}
	if result.CreateEvaluator.Evaluator == nil {
		return "", fmt.Errorf("failed to create evaluator: empty response")
	}
	return result.CreateEvaluator.Evaluator.ID, nil
}

func (c *Client) EditEvaluator(ctx context.Context, evaluatorID, name, description string) error {
	const query = `
mutation EditEvaluator($input: EditEvaluatorMutationInput!) {
  editEvaluator(input: $input) { evaluator { id } }
}`
	input := map[string]interface{}{
		"evaluatorId": evaluatorID,
		"name":        name,
		"description": description,
	}
	var result map[string]interface{}
	if err := c.execute(ctx, query, map[string]interface{}{"input": input}, &result); err != nil {
		return fmt.Errorf("failed to edit evaluator: %w", err)
	}
	return nil
}

func (c *Client) CreateEvaluatorVersion(ctx context.Context, evaluatorID, commitMessage string, config EvaluatorConfigInput) error {
	const query = `
mutation CreateEvaluatorVersion($input: CreateEvaluatorVersionMutationInput!) {
  createEvaluatorVersion(input: $input) { evaluatorVersion { id } }
}`
	input := map[string]interface{}{
		"evaluatorId":   evaluatorID,
		"commitMessage": commitMessage,
	}
	config.apply(input)

	var result map[string]interface{}
	if err := c.execute(ctx, query, map[string]interface{}{"input": input}, &result); err != nil {
		return fmt.Errorf("failed to create evaluator version: %w", err)
	}
	return nil
}

func (c *Client) DeleteEvaluator(ctx context.Context, evaluatorID string) error {
	const query = `
mutation DeleteEvaluator($input: DeleteEvaluatorMutationInput!) {
  deleteEvaluator(input: $input) { success }
}`
	var result struct {
		DeleteEvaluator struct {
			Success bool `json:"success"`
		} `json:"deleteEvaluator"`
	}
	if err := c.execute(ctx, query, map[string]interface{}{"input": map[string]interface{}{"evaluatorId": evaluatorID}}, &result); err != nil {
		return fmt.Errorf("failed to delete evaluator: %w", err)
	}
	if !result.DeleteEvaluator.Success {
		return fmt.Errorf("failed to delete evaluator %s: API reported failure", evaluatorID)
	}
	return nil
}
