# Finetuning Gemini

A Go application for fine-tuning and interacting with Google's Gemini AI model, featuring a web interface built with Fiber and Templ.

## Features

- Web-based interface for interacting with Gemini AI
- Fine-tuning capabilities for Gemini models
- Sample data generation and validation
- Chat interface for model interaction
- Modern UI built with Templ components

## Prerequisites

- Go 1.23.0 or later
- Docker (optional, for containerized deployment)

## Installation

### Local Development

1. Clone the repository:
```bash
git clone https://github.com/EthicalGopher/finetuning-gemini.git
cd finetuning-gemini
```

2. Install dependencies:
```bash
go mod download
```

3. Run the application:
```bash
go run main.go
```

The application will be available at `http://localhost:3000`

### Docker Deployment

1. Build the Docker image:
```bash
docker build -t finetuning-gemini .
```

2. Run the container:
```bash
docker run -p 3000:3000 finetuning-gemini
```

## API Endpoints

- `GET /` - Main application interface
- `GET /chat` - Chat interface
- `POST /api/finetune` - Fine-tune the model
- `GET /api/sample` - Get sample data
- `POST /api/validate` - Validate data
- `POST /api/chat` - Chat with the model

## Project Structure

- `main.go` - Main application entry point
- `template/` - Templ components for the web interface
- `Dockerfile` - Container configuration
- `go.mod` - Go module dependencies

## Dependencies

- [Fiber](https://github.com/gofiber/fiber) - Web framework
- [Templ](https://github.com/a-h/templ) - HTML templating
- [Google Generative AI](https://github.com/google/generative-ai-go) - Gemini API client
