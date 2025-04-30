package main

import (
	"context"
	templates "github.com/EthicalGopher/finetuning-gemini/template"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func server() {
	app := fiber.New()
	defer app.Listen(":3000")
	app.Use(cors.New(cors.Config{
		AllowMethods: "GET,POST",
	}))
	app.Get("/", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		component := templates.Index()
		return component.Render(context.Background(), c)
	})
	app.Get("/chat", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		component := templates.Chat()
		return component.Render(context.Background(), c)
	})
	api := app.Group("api")
	api.Post("/finetune", handleTuning)
	api.Get("/sample", handleSampleData)
	api.Post("/validate", handleValidata)
	api.Post("/chat", handleChat)
}
func main() {
	server()
}
