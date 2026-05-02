// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package cmd

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestParseOllamaURL_Internal(t *testing.T) {
	tests := []struct {
		name         string
		ollamaURL    string
		expectedHost string
		expectedPort int
	}{
		{"empty", "", "localhost", 11434},
		{"host", "1.2.3.4", "1.2.3.4", 11434},
		{"host-port", "1.2.3.4:5678", "1.2.3.4", 5678},
		{"http", "http://ollama:11434", "ollama", 11434},
		{"https", "https://ollama", "ollama", 11434},
		{"invalid-port", "localhost:abc", "localhost", 11434},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := ParseOllamaURL(tt.ollamaURL)
			assert.Equal(t, tt.expectedHost, host)
			assert.Equal(t, tt.expectedPort, port)
		})
	}
}

func TestWorkflowNeedsOllama_Internal(t *testing.T) {
	tests := []struct {
		name     string
		workflow *domain.Workflow
		expected bool
	}{
		{
			"no resources",
			&domain.Workflow{Resources: []*domain.Resource{}},
			false,
		},
		{
			"ollama resource",
			&domain.Workflow{
				Resources: []*domain.Resource{
					{
						Run: domain.RunConfig{
							Chat: &domain.ChatConfig{Backend: "ollama"},
						},
					},
				},
			},
			true,
		},
		{
			"default backend resource",
			&domain.Workflow{
				Resources: []*domain.Resource{
					{
						Run: domain.RunConfig{
							Chat: &domain.ChatConfig{Backend: ""},
						},
					},
				},
			},
			true,
		},
		{
			"non-ollama resource",
			&domain.Workflow{
				Resources: []*domain.Resource{
					{
						Run: domain.RunConfig{
							Chat: &domain.ChatConfig{Backend: "openai"},
						},
					},
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, workflowNeedsOllama(tt.workflow))
		})
	}
}

func TestIsOllamaRunning_Internal(t *testing.T) {
	assert.False(t, IsOllamaRunning("127.0.0.1", 0))

	// IsOllamaRunning checks TCP connectivity only - any listening port returns true
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port
	assert.True(t, IsOllamaRunning("127.0.0.1", port))
}

func TestResolveWorkflowPath_Internal(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("absolute path file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "workflow.yaml")
		_ = os.WriteFile(path, []byte("test"), 0644)
		resolved, cleanup, err := resolveWorkflowPath(path)
		require.NoError(t, err)
		assert.Equal(t, path, resolved)
		assert.Nil(t, cleanup)
	})

	t.Run("directory", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "mydir")
		_ = os.Mkdir(dir, 0755)
		path := filepath.Join(dir, "workflow.yaml")
		_ = os.WriteFile(path, []byte("test"), 0644)
		resolved, cleanup, err := resolveWorkflowPath(dir)
		require.NoError(t, err)
		assert.Equal(t, path, resolved)
		assert.Nil(t, cleanup)
	})

	t.Run("package", func(t *testing.T) {
		// This tests the ExtractPackage path
		// We'll just mock a .kdeps file
		pkg := filepath.Join(tmpDir, "test.kdeps")
		_ = os.WriteFile(pkg, []byte("invalid content"), 0644)
		_, _, err := resolveWorkflowPath(pkg)
		assert.Error(t, err) // Should fail to extract
	})
}

func TestEnsureOllamaRunning_Internal(t *testing.T) {
	// This function is complex, let's test some branches

	t.Run("already running", func(_ *testing.T) {
		// Mock IsOllamaRunning to return true if possible?
		// Not easy without mocking the network.
	})

	t.Run("command not found", func(t *testing.T) {
		// Find a free port so IsOllamaRunning returns false (not already running)
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		port := l.Addr().(*net.TCPAddr).Port
		l.Close() // close immediately so the port is free but not listening

		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", "")
		defer os.Setenv("PATH", oldPath)

		ollamaURL := fmt.Sprintf("http://127.0.0.1:%d", port)
		err = ensureOllamaRunning(ollamaURL)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ollama not found in PATH")
	})
}

func TestPrintIORequirements_Internal(_ *testing.T) {
	// Just ensure it doesn't panic with various configs
	w := &domain.Workflow{
		Resources: []*domain.Resource{
			{Run: domain.RunConfig{APIResponse: &domain.APIResponseConfig{}}},
		},
	}
	printIORequirements(w)
}

func TestFindWorkflowFile_Internal(t *testing.T) {
	tmpDir := t.TempDir()

	path := FindWorkflowFile(tmpDir)
	assert.Equal(t, "", path)

	wPath := filepath.Join(tmpDir, "workflow.yaml")
	_ = os.WriteFile(wPath, []byte("test"), 0644)
	path = FindWorkflowFile(tmpDir)
	assert.Equal(t, wPath, path)
}

// TestPrintIORequirements_WithNonAPIInput exercises the non-trivial branches of
// printIORequirements: bot sources, capture sources, transcriber, and activation.
func TestPrintIORequirements_WithNonAPIInput(t *testing.T) {
	// audio source → exercises printCaptureRequirements (audio branch)
	t.Run("audio_source", func(_ *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{
					Sources: []string{domain.InputSourceAudio},
				},
			},
		}
		printIORequirements(w)
	})

	// video source → exercises printCaptureRequirements (video branch)
	t.Run("video_source", func(_ *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{
					Sources: []string{domain.InputSourceVideo},
				},
			},
		}
		printIORequirements(w)
	})

	// bot source with all four platforms
	t.Run("bot_source_all_platforms", func(_ *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{
					Sources: []string{domain.InputSourceBot},
					Bot: &domain.BotConfig{
						Discord:  &domain.DiscordConfig{BotToken: "tok"},
						Slack:    &domain.SlackConfig{BotToken: "tok"},
						Telegram: &domain.TelegramConfig{BotToken: "tok"},
						WhatsApp: &domain.WhatsAppConfig{PhoneNumberID: "id"},
					},
				},
			},
		}
		printIORequirements(w)
	})

	// offline transcriber with whisper engine
	t.Run("transcriber_offline_whisper", func(_ *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{
					Sources: []string{domain.InputSourceAudio},
					Transcriber: &domain.TranscriberConfig{
						Mode: domain.TranscriberModeOffline,
						Offline: &domain.OfflineTranscriberConfig{
							Engine: domain.TranscriberEngineWhisper,
						},
					},
				},
			},
		}
		printIORequirements(w)
	})

	// offline transcriber with faster-whisper engine
	t.Run("transcriber_offline_faster_whisper", func(_ *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{
					Sources: []string{domain.InputSourceAudio},
					Transcriber: &domain.TranscriberConfig{
						Mode: domain.TranscriberModeOffline,
						Offline: &domain.OfflineTranscriberConfig{
							Engine: domain.TranscriberEngineFasterWhisper,
						},
					},
				},
			},
		}
		printIORequirements(w)
	})

	// offline transcriber with vosk engine
	t.Run("transcriber_offline_vosk", func(_ *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{
					Sources: []string{domain.InputSourceAudio},
					Transcriber: &domain.TranscriberConfig{
						Mode: domain.TranscriberModeOffline,
						Offline: &domain.OfflineTranscriberConfig{
							Engine: domain.TranscriberEngineVosk,
						},
					},
				},
			},
		}
		printIORequirements(w)
	})

	// offline transcriber with whisper-cpp engine
	t.Run("transcriber_offline_whispercpp", func(_ *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{
					Sources: []string{domain.InputSourceAudio},
					Transcriber: &domain.TranscriberConfig{
						Mode: domain.TranscriberModeOffline,
						Offline: &domain.OfflineTranscriberConfig{
							Engine: domain.TranscriberEngineWhisperCPP,
						},
					},
				},
			},
		}
		printIORequirements(w)
	})

	// offline activation with whisper engine (exercises printActivationRequirements)
	t.Run("activation_offline_whisper", func(_ *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{
					Sources: []string{domain.InputSourceAudio},
					Activation: &domain.ActivationConfig{
						Mode: domain.TranscriberModeOffline,
						Offline: &domain.OfflineTranscriberConfig{
							Engine: domain.TranscriberEngineWhisper,
						},
					},
				},
			},
		}
		printIORequirements(w)
	})

	// duplicate engine in transcriber + activation → printed[engine] dedup
	t.Run("dedup_printed_engines", func(_ *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{
					Sources: []string{domain.InputSourceAudio},
					Transcriber: &domain.TranscriberConfig{
						Mode: domain.TranscriberModeOffline,
						Offline: &domain.OfflineTranscriberConfig{
							Engine: domain.TranscriberEngineWhisper,
						},
					},
					Activation: &domain.ActivationConfig{
						Mode: domain.TranscriberModeOffline,
						Offline: &domain.OfflineTranscriberConfig{
							Engine: domain.TranscriberEngineWhisper, // same engine → skipped
						},
					},
				},
			},
		}
		printIORequirements(w)
	})

	// nil transcriber / nil activation → early return
	t.Run("nil_transcriber_activation", func(_ *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{
					Sources:     []string{domain.InputSourceAudio},
					Transcriber: nil,
					Activation:  nil,
				},
			},
		}
		printIORequirements(w)
	})

	// online transcriber → printTranscriberRequirements returns early (mode != offline)
	t.Run("transcriber_online", func(_ *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{
					Sources: []string{domain.InputSourceAudio},
					Transcriber: &domain.TranscriberConfig{
						Mode: "online",
					},
				},
			},
		}
		printIORequirements(w)
	})
}

// TestWaitForOllamaReady_Timeout verifies that waitForOllamaReady returns an
// error when Ollama does not start within the timeout window.
func TestWaitForOllamaReady_Timeout(t *testing.T) {
	// Use port 0 (invalid) so IsOllamaRunning always returns false.
	err := waitForOllamaReady("127.0.0.1", 0, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

// TestEnsureOllamaRunning_AlreadyRunning verifies the "already running" path by
// starting a TCP listener so IsOllamaRunning returns true.
func TestEnsureOllamaRunning_AlreadyRunning(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port
	ollamaURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	err = ensureOllamaRunning(ollamaURL)
	assert.NoError(t, err)
}
