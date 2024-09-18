package main

import (
	"context"
	"fmt"
	"kdeps/pkg/docker"
	"kdeps/pkg/logging"
	"kdeps/pkg/resolver"
	"log"
	"os"

	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms/ollama"
)

func main() {
	var apiServerMode bool
	// Create an afero filesystem (you can use afero.NewOsFs() for the real filesystem)
	fs := afero.NewOsFs()
	logger := logging.GetLogger()

	// Check if /.dockerenv exists
	exists, err := afero.Exists(fs, "/.dockerenv")
	if err != nil {
		logging.Error("Error checking /.dockerenv existence: ", err)
		log.Fatal(err)
	}

	if exists {
		dr, err := resolver.NewGraphResolver(fs, logger, "/agent/workflow/")
		if err != nil {
			log.Fatal(err)
		}

		// Call BootstrapDockerSystem to initialize Docker and pull models
		apiServerMode, err = docker.BootstrapDockerSystem(fs, dr)
		if err != nil {
			fmt.Printf("Error during bootstrap: %v\n", err)
			os.Exit(1) // Exit with a non-zero status on failure
		}
	}

	logging.Info("Bootstrap completed successfully.")

	llm, err := ollama.New(ollama.WithModel("tinyllama"))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	completion, err := llm.Call(ctx, "Human: Who was the first man to walk on the moon?\nAssistant:")
	if err != nil {
		log.Fatal(err)
	}

	logging.Info("completion: ", completion)

	if apiServerMode {
		select {}
	}

	// llm, err = ollama.New(ollama.WithModel("tinydolphin"))
	// if err != nil {
	//	log.Fatal(err)
	// }

	// completion, err = llm.Call(ctx, fmt.Sprintf("OK: Tinyllama said '%s', is this true? Anything to add?", completion))
	// if err != nil {
	//	log.Fatal(err)
	// }

}
