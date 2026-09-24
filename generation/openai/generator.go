package openai

import (
	"context"
	"fmt"
	"strings"

	"rag-document-assistant/helper"
	"rag-document-assistant/services"
)

const groundedInstructions = `You are a document question-answering assistant.
Use only the supplied context to answer the question.
Do not use outside knowledge.
Do not invent or infer unsupported facts.
If the context does not contain sufficient evidence, reply exactly:
I don't have enough information to answer that.
Keep the answer concise and do not add a sources section; the application adds citations separately.`

type Generator struct {
	client *helper.OpenAIClient
	model  string
}

func New(client *helper.OpenAIClient, model string) *Generator {
	return &Generator{client: client, model: model}
}

func (g *Generator) Generate(ctx context.Context, question, contextText string) (string, error) {
	if strings.TrimSpace(contextText) == "" {
		return services.InsufficientInformation, nil
	}
	input := fmt.Sprintf("CONTEXT:\n%s\n\nQUESTION:\n%s", contextText, question)
	var response responseBody
	if err := g.client.Post(ctx, "/responses", responseRequest{
		Model: g.model, Instructions: groundedInstructions, Input: input, Store: false,
	}, &response); err != nil {
		return "", err
	}
	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Type == "output_text" && strings.TrimSpace(content.Text) != "" {
				return strings.TrimSpace(content.Text), nil
			}
		}
	}
	return "", fmt.Errorf("OpenAI response contained no output text")
}

type responseRequest struct {
	Model        string `json:"model"`
	Instructions string `json:"instructions"`
	Input        string `json:"input"`
	Store        bool   `json:"store"`
}

type responseBody struct {
	Output []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}
