package main

import (
	"context"
	"fmt"
	"kdeps/pkg/docker"
	"kdeps/pkg/logging"
	"log"
	"os"

	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms/ollama"
)

func main() {
	// Create an afero filesystem (you can use afero.NewOsFs() for the real filesystem)
	fs := afero.NewOsFs()

	// Call BootstrapDockerSystem to initialize Docker and pull models
	err := docker.BootstrapDockerSystem(fs)
	if err != nil {
		fmt.Printf("Error during bootstrap: %v\n", err)
		os.Exit(1) // Exit with a non-zero status on failure
	}

	logging.Info("Bootstrap completed successfully.")

	llm, err := ollama.New(ollama.WithModel("llama3.1"))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	completion, err := llm.Call(ctx, "Human: Who was the first man to walk on the moon?\nAssistant:")
	if err != nil {
		log.Fatal(err)
	}

	logging.Info("completion: ", completion)

}
