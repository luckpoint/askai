package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/koki-develop/askai/internal/config"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/net/context"
)

var (
	youHeader = lipgloss.NewStyle().Background(lipgloss.Color("#00ADD8")).Foreground(lipgloss.Color("#000000")).Padding(0, 1).Render("You")
)

type UI struct {
	writer          io.Writer
	client          *openai.Client
	model           string
	interactive     bool
	question        *string
	messages        []openai.ChatCompletionMessage
	system_messages map[string]config.SystemMessage
}

type Config struct {
	APIKey         string
	Model          string
	Interactive    bool
	Question       *string
	Messages       []openai.ChatCompletionMessage
	SystemMessages map[string]config.SystemMessage
}

func New(cfg *Config) *UI {
	client := openai.NewClient(cfg.APIKey)

	return &UI{
		writer:          os.Stdout,
		client:          client,
		model:           cfg.Model,
		interactive:     cfg.Interactive,
		question:        cfg.Question,
		messages:        cfg.Messages,
		system_messages: cfg.SystemMessages,
	}
}

func (ui *UI) Start() error {
	ctx := context.Background()

	if !ui.interactive && ui.question == nil {
		return errors.New("question is required when interactive mode is disabled")
	}

	var currentTag string
	for {
		var msg string

		if ui.question == nil {
			ipt, ok, err := ui.readInput()
			if err != nil {
				return err
			}
			if !ok {
				break
			}
			msg = ipt
		} else {
			msg = *ui.question
			ui.question = nil
		}

		// Check if the message starts with "@"
		if strings.HasPrefix(msg, "@") {
			// Retrieve the first word of the message that starts with "@"
			tag := strings.Fields(msg)[0]
			currentTag = tag // Record the current tag
			if systemMessage, ok := ui.system_messages[tag]; ok {
				// Add the corresponding system message
				ui.messages = append(ui.messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemMessage.Content,
				})

				// Remove the tag part (first word) from the message and treat the rest as a user message
				msg = strings.TrimSpace(strings.TrimPrefix(msg, tag))
			}
		}

		ui.messages = append(ui.messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: msg})
		if ui.interactive {
			_, _ = ui.writer.Write([]byte(youHeader))
			_, _ = ui.writer.Write([]byte{'\n'})
			_, _ = ui.writer.Write([]byte(strings.TrimSpace(msg)))
			_, _ = ui.writer.Write([]byte{'\n', '\n'})
		}

		// Modify aiHeader when displaying AI's response
		ans, err := ui.printAnswer(ctx, currentTag)
		if err != nil {
			return err
		}

		if !ui.interactive {
			break
		}

		ui.messages = append(ui.messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, Content: ans})
	}

	return nil
}

func (ui *UI) readInput() (string, bool, error) {
	m := newInputModel()
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return "", false, err
	}
	if m.abort {
		return "", false, nil
	}
	return m.value, true, nil
}

func (ui *UI) printAnswer(ctx context.Context, tag string) (string, error) {
	// Change aiHeader according to the tag
	aiHeaderContent := "AI"
	if tag != "@Default" {
		aiHeaderContent = fmt.Sprintf("AI (%s)", tag)
	}
	aiHeader := lipgloss.NewStyle().Background(lipgloss.Color("#ffffff")).Foreground(lipgloss.Color("#000000")).Padding(0, 1).Render(aiHeaderContent)

	stream, err := ui.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Messages: ui.messages,
		Model:    ui.model,
		Stream:   true,
	})
	if err != nil {
		return "", err
	}
	defer stream.Close()

	b := new(strings.Builder)
	if ui.interactive {
		_, _ = ui.writer.Write([]byte(aiHeader))
		_, _ = ui.writer.Write([]byte{'\n'})
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
		s := resp.Choices[0].Delta.Content
		b.WriteString(s)
		_, _ = ui.writer.Write([]byte(s))
	}
	if ui.interactive {
		_, _ = ui.writer.Write([]byte{'\n'})
	}
	_, _ = ui.writer.Write([]byte{'\n'})

	return b.String(), nil
}

// Function to output the Config structure to the standard output
func (cfg *Config) Print() {
	fmt.Printf("APIKey: %s\n", cfg.APIKey)
	fmt.Printf("Model: %s\n", cfg.Model)
	fmt.Printf("Messages: %+v\n", cfg.Messages)
	fmt.Printf("SystemMessages: %+v\n", cfg.SystemMessages)
}
