package generator

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var (
	flagModel = flag.String("model", "gpt-5-nano-ca", "The model to use")
	flagEnv   = flag.String("env", ".env", "Path to .env file")
)

func main() {
	flag.Parse()
	// load environment variables from .env file
	err := godotenv.Load(*flagEnv)
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")
	apiKey := os.Getenv("OPENAI_API_KEY")

	// query llm
	ctx := context.Background()
	llm, err := openai.New(
		openai.WithBaseURL(baseURL),
		openai.WithToken(apiKey),
		openai.WithModel(*flagModel),
	)
	if err != nil {
		log.Fatal(err)
	}
	prompt := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "Who are you?"),
	}
	response, err := llm.GenerateContent(ctx, prompt)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Choices[0].Content)
}
