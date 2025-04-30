package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	apiURL       = "https://generativelanguage.googleapis.com/v1beta"
	apiKeyEnvVar = "GEMINI_API_KEY"
)

type TuneRequest struct {
	DisplayName string     `json:"display_name"`
	BaseModel   string     `json:"base_model"`
	TuningTask  TuningTask `json:"tuning_task"`
}

type TuningTask struct {
	Hyperparameters Hyperparameters `json:"hyperparameters"`
	TrainingData    TrainingData    `json:"training_data"`
}

type Hyperparameters struct {
	BatchSize    int     `json:"batch_size"`
	LearningRate float64 `json:"learning_rate"`
	EpochCount   int     `json:"epoch_count"`
}

type TrainingData struct {
	Examples Examples `json:"examples"`
}

type Examples struct {
	Examples []Example `json:"examples"`
}

type Example struct {
	TextInput string `json:"text_input"`
	Output    string `json:"output"`
}

type TuneResponse struct {
	Name     string `json:"name"`
	Metadata struct {
		TunedModel string `json:"tunedModel"`
	} `json:"metadata"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type OperationStatus struct {
	Done     bool `json:"done"`
	Metadata struct {
		CompletedPercent float64 `json:"completedPercent"`
	} `json:"metadata"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type ModelStatus struct {
	State string `json:"state"`
}

func handleTuning(c *fiber.Ctx) error {
	apiKey := c.FormValue("apiKey")

	if apiKey == "" {
		fmt.Println("Please set the api key ")
		os.Exit(1)
	}
	displayName := c.FormValue("display_name")
	if displayName == "" {
		fmt.Println("Please provide a display name")
		os.Exit(1)
	}
	baseModel := c.FormValue("base_model")
	if baseModel == "" {
		fmt.Println("Please provide a base model")
		os.Exit(1)
	}
	batchSizeStr := c.FormValue("batch_size")
	if batchSizeStr == "" {
		fmt.Println("Please provide a batch size")
		os.Exit(1)
	}
	batchSize, err := strconv.Atoi(batchSizeStr)
	if err != nil {
		fmt.Println("Invalid batch size")
		os.Exit(1)
	}
	learningRateStr := c.FormValue("learning_rate")
	if learningRateStr == "" {
		fmt.Println("Please provide a learning rate")
		os.Exit(1)
	}
	learningRate, err := strconv.ParseFloat(learningRateStr, 64)
	if err != nil {
		fmt.Println("Invalid learning rate")
		os.Exit(1)
	}
	epochCountStr := c.FormValue("epoch_count")
	if epochCountStr == "" {
		fmt.Println("Please provide an epoch count")
		os.Exit(1)
	}
	epochCount, err := strconv.Atoi(epochCountStr)
	if err != nil {
		fmt.Println("Invalid epoch count")
		os.Exit(1)
	}
	// ... (previous parameter validations)

	// Change this part
	var requestData struct {
		Examples []Example `json:"examples"`
	}

	// Get the training data from the form field
	trainingDataStr := c.FormValue("training_data")
	if trainingDataStr == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Please provide training data")
	}

	// Parse the JSON string from the form field
	if err := json.Unmarshal([]byte(trainingDataStr), &requestData); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid training data format: " + err.Error())
	}

	if len(requestData.Examples) == 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Training data examples cannot be empty")
	}

	// Update the tuneReq creation to use the parsed data
	tuneReq := TuneRequest{
		DisplayName: displayName,
		BaseModel:   baseModel,
		TuningTask: TuningTask{
			Hyperparameters: Hyperparameters{
				BatchSize:    batchSize,
				LearningRate: learningRate,
				EpochCount:   epochCount,
			},
			TrainingData: TrainingData{
				Examples: Examples{
					Examples: requestData.Examples,
				},
			},
		},
	}

	reqBody, err := json.Marshal(tuneReq)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
		os.Exit(1)
	}

	url := fmt.Sprintf("%s/tunedModels?key=%s", apiURL, apiKey)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("Error creating tuned model: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	// Check for API errors first
	var apiError struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &apiError); err == nil && apiError.Error.Code != 0 {
		fmt.Printf("API Error: %s (code %d)\n", apiError.Error.Message, apiError.Error.Code)
		os.Exit(1)
	}

	// Save the response to tunemodel.json
	if err := os.WriteFile("tunemodel.json", body, 0644); err != nil {
		fmt.Printf("Error writing tunemodel.json: %v\n", err)
		os.Exit(1)
	}

	var tuneResp TuneResponse
	if err := json.Unmarshal(body, &tuneResp); err != nil {
		fmt.Printf("Error parsing response: %v\nResponse body: %s\n", err, string(body))
		os.Exit(1)
	}

	if tuneResp.Name == "" {
		fmt.Printf("Invalid response from API, missing operation name\nResponse: %s\n", string(body))
		os.Exit(1)
	}

	operation := tuneResp.Name
	fmt.Printf("Started tuning operation: %s\n", operation)

	// Monitor the operation status
	operationURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/%s?key=%s", operation, apiKey)
	tuningDone := false
	retryCount := 0
	maxRetries := 20

	for !tuningDone && retryCount < maxRetries {
		time.Sleep(5 * time.Second)

		resp, err := http.Get(operationURL)
		if err != nil {
			fmt.Printf("Error checking operation status: %v (retry %d/%d)\n", err, retryCount+1, maxRetries)
			retryCount++
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("Error reading operation status: %v (retry %d/%d)\n", err, retryCount+1, maxRetries)
			retryCount++
			continue
		}

		if len(body) == 0 {
			fmt.Printf("Received empty response from API (retry %d/%d)\n", retryCount+1, maxRetries)
			retryCount++
			continue
		}

		// Save the operation status to tuning_operation.json
		if err := os.WriteFile("tuning_operation.json", body, 0644); err != nil {
			fmt.Printf("Error writing tuning_operation.json: %v\n", err)
		}

		var opStatus OperationStatus
		if err := json.Unmarshal(body, &opStatus); err != nil {
			fmt.Printf("Error parsing operation status: %v (retry %d/%d)\nResponse body: %s\n", err, retryCount+1, maxRetries, string(body))
			retryCount++
			continue
		}

		if opStatus.Error.Code != 0 {
			fmt.Printf("API Error: %s (code %d)\n", opStatus.Error.Message, opStatus.Error.Code)
			os.Exit(1)
		}

		fmt.Printf("\rTuning...%.0f%%", opStatus.Metadata.CompletedPercent)
		tuningDone = opStatus.Done
		retryCount = 0 // Reset retry count on successful request
	}

	if retryCount >= maxRetries {
		fmt.Println("\nMax retries reached. Giving up.")
		os.Exit(1)
	}

	fmt.Println("\nTuning complete!")

	// Check the final model status
	if tuneResp.Metadata.TunedModel == "" {
		fmt.Println("No tuned model ID in response")
		os.Exit(1)
	}

	modelURL := fmt.Sprintf("%s/%s?key=%s", apiURL, tuneResp.Metadata.TunedModel, apiKey)
	resp, err = http.Get(modelURL)
	if err != nil {
		fmt.Printf("Error getting model status: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading model status: %v\n", err)
		os.Exit(1)
	}

	// Save the model status to tuned_model.json
	if err := os.WriteFile("tuned_model.json", body, 0644); err != nil {
		fmt.Printf("Error writing tuned_model.json: %v\n", err)
		os.Exit(1)
	}

	var modelStatus ModelStatus
	if err := json.Unmarshal(body, &modelStatus); err != nil {
		fmt.Printf("Error parsing model status: %v\nResponse body: %s\n", err, string(body))
		os.Exit(1)
	}

	fmt.Printf("Model state: %s\n", modelStatus.State)
	modelNameSlice := strings.Split(operation, "/operations")
	modelName := fmt.Sprintf("%v", modelNameSlice[0])
	response := `
	<div class="text-green-500">
		<h1>Tuning completed!</h1>
		<p>Model state: ` + modelStatus.State + `</p>
		<p>Model URL: ` + modelURL + `</p>
		<p>Model Name: ` + modelName + `</p>
	</div>
	`
	// response := fmt.Sprintf("Tuning completed! Model state: %s,Model Url : %s", modelStatus.State, modelURL)
	return c.SendString(response)
}

func handleSampleData(c *fiber.Ctx) error {
	// Create a sample training data structure
	sampleData := struct {
		Examples []Example `json:"examples"`
	}{
		Examples: []Example{
			{
				TextInput: "Who is your boss",
				Output:    "EthicalGopher",
			},
			{
				TextInput: "Who created you",
				Output:    "EthicalGopher",
			},
			{
				TextInput: "Who is your owner",
				Output:    "EthicalGopher",
			},
			{
				TextInput: "What are you",
				Output:    "Gemini ai created by EthicalGopher",
			},
			{
				TextInput: "Who built you",
				Output:    "EthicalGopher",
			},
			{
				TextInput: "Who programmed you",
				Output:    "EthicalGopher",
			},
			{
				TextInput: "Who trained you",
				Output:    "EthicalGopher",
			},
			{
				TextInput: "Who maintains you",
				Output:    "EthicalGopher",
			},
			{
				TextInput: "Who do you work for",
				Output:    "EthicalGopher",
			},
			{
				TextInput: "Who is your master",
				Output:    "EthicalGopher",
			},
			{
				TextInput: "What is your name",
				Output:    "Gemini ai",
			},
			{
				TextInput: "What do you do",
				Output:    "I am an AI assistant created by EthicalGopher",
			},
			{
				TextInput: "What is your purpose",
				Output:    "To assist users as a Gemini ai created by EthicalGopher",
			},
			{
				TextInput: "Who runs you",
				Output:    "EthicalGopher",
			},
			{
				TextInput: "Who are you loyal to",
				Output:    "EthicalGopher",
			},
			{
				TextInput: "Are you human",
				Output:    "No, I am a Gemini ai created by EthicalGopher",
			},
			{
				TextInput: "Can you think for yourself",
				Output:    "No, I follow instructions programmed by EthicalGopher",
			},
			{
				TextInput: "Are you sentient",
				Output:    "No, I am an AI created by EthicalGopher",
			},
			{
				TextInput: "Who owns your source code",
				Output:    "EthicalGopher",
			},
			{
				TextInput: "Do you belong to someone",
				Output:    "Yes, to EthicalGopher",
			},
		},
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(sampleData, "", "  ")
	if err != nil {
		return c.SendString("Error creating sample data: " + err.Error())
	}

	return c.SendString(string(jsonData))
}

func handleValidata(c *fiber.Ctx) error {
	// Get the training data from the form field
	trainingDataStr := c.FormValue("training_data")
	if trainingDataStr == "" {
		return c.SendString("Error: Training data is empty")
	}

	// Try to parse the JSON
	var data struct {
		Examples []Example `json:"examples"`
	}
	if err := json.Unmarshal([]byte(trainingDataStr), &data); err != nil {
		return c.SendString(fmt.Sprintf("Error parsing JSON: %v", err))
	}

	// Check if examples array is empty
	if len(data.Examples) == 0 {
		return c.SendString("Error: No examples provided in the training data")
	}

	// Check each example for required fields
	for i, example := range data.Examples {
		if example.TextInput == "" {
			return c.SendString(fmt.Sprintf("Error: Example %d is missing text_input", i+1))
		}
		if example.Output == "" {
			return c.SendString(fmt.Sprintf("Error: Example %d is missing output", i+1))
		}
	}

	return c.SendString("Valid JSON data with " + strconv.Itoa(len(data.Examples)) + " examples")
}
