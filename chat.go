package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type ChatRequest struct {
	ModelName string `json:"modelName"`
	ApiKey    string `json:"apiKey"`
	Input     string `json:"input"`
}

func handleChat(c *fiber.Ctx) error {
	var request ChatRequest
	if err := c.BodyParser(&request); err != nil {
		fmt.Printf("Error parsing request: %v\n", err)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request format")
	}

	// Log the request for debugging

	response, err := Generateresponse(request)
	if err != nil {
		fmt.Printf("Error generating response: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error generating response: " + err.Error())
	}

	// Return the response as plain text
	return c.SendString(response)
}

func Generateresponse(ai ChatRequest) (string, error) {

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(ai.ApiKey))
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(ai.ModelName)

	resp, err := model.GenerateContent(ctx, genai.Text(ai.Input))
	if err != nil {
		return "", err
	}

	// Check if there are any candidates in the response
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no response generated from the model")
	}

	var responseText strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			responseText.WriteString(string(text))
		}
	}

	return responseText.String(), nil
}
