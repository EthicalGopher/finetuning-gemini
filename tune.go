package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
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
	if err := c.BodyParser(&requestData); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Please provide training data")
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
	response := fmt.Sprintf("Tuning completed! Model state: %s,Model Name : %s", modelStatus.State, modelURL)
	return c.SendString(response)
}

func handleSampleData(c *fiber.Ctx) error {
	response := []map[string]string{
		{
			"text_input": "What is the basic pattern for error handling in Go?",
			"output":     "The basic error handling pattern in Go is to return an error as the last return value and check it using: if err != nil { return err }. Unlike other languages, Go doesn't use exceptions but explicit error checking through return values.",
		},
		{
			"text_input": "How do you create a custom error in Go?",
			"output":     "In Go, you can create custom errors in several ways: 1) Using errors.New('message') 2) Using fmt.Errorf('formatted %s', value) 3) Creating a custom type that implements the error interface by defining an Error() string method.",
		},
		{
			"text_input": "What is error wrapping in Go?",
			"output":     "Error wrapping in Go (introduced in Go 1.13) allows you to add context to errors while preserving the original error. You can wrap errors using fmt.Errorf('additional context: %w', err). The %w verb creates a wrapped error that can be unwrapped using errors.Unwrap().",
		},
		{
			"text_input": "How do you check for specific error types in Go?",
			"output":     "In Go, you can check for specific error types using: 1) errors.Is() to check if an error equals a specific error value 2) errors.As() to check if an error is of a specific type 3) Type assertions if you need to access methods of a specific error type. Example: if errors.Is(err, io.EOF) { // handle EOF }",
		},
		{
			"text_input": "What is the difference between panic and error in Go?",
			"output":     "In Go, errors are values that indicate something went wrong but can be handled, while panic is for unrecoverable situations. Errors are part of normal flow control and should be explicitly checked. Panic stops normal execution, runs deferred functions, and crashes the program unless recovered.",
		},
		{
			"text_input": "How do you implement the error interface in Go?",
			"output":     "To implement the error interface in Go, define a type and implement the Error() string method. Example: type MyError struct { Message string } func (e *MyError) Error() string { return e.Message }. This type now satisfies the error interface and can be used wherever an error is expected.",
		},
		{
			"text_input": "What is error handling with multiple return values in Go?",
			"output":     "In Go, functions often return a result and an error: func DoSomething() (Result, error). The conventional pattern is to check the error first: result, err := DoSomething(); if err != nil { return nil, err }. If no error occurs, proceed with using the result.",
		},
		{
			"text_input": "How do you use defer and recover for error handling in Go?",
			"output":     "Defer and recover are used for panic handling in Go. Place a deferred function that calls recover() to catch panics: defer func() { if r := recover(); r != nil { // handle panic } }(). This pattern prevents program crashes and allows graceful error handling of panics.",
		},
		{
			"text_input": "What are sentinel errors in Go?",
			"output":     "Sentinel errors in Go are predefined error values that can be used for comparison. Common examples include io.EOF or sql.ErrNoRows. They are created using var ErrXxx = errors.New('message') and compared using errors.Is(err, ErrXxx). However, they should be used sparingly as they create dependencies.",
		},
		{
			"text_input": "How do you handle multiple errors in Go?",
			"output":     "In Go, multiple errors can be handled by: 1) Using error wrapping to chain errors 2) Creating custom error types that contain multiple errors 3) Using packages like 'go.uber.org/multierr' or 'hashicorp/go-multierror' to combine multiple errors into a single error value. Example: if err1 != nil { return fmt.Errorf('operation failed: %w', err1) }",
		},
	}
	return c.JSON(response)
}

func handleValidata(c *fiber.Ctx) error {
	var data struct {
		Examples []Example `json:"examples"`
	}
	if err := c.BodyParser(&data); err != nil {
		return c.SendString(fmt.Sprintf("Error parsing JSON: %v", err))
	}
	return c.SendString("Valid JSON data")
}
