package cmd

import (
	"bytes"
	"errors"
	"os"
	"strings"

	"github.com/koki-develop/askai/internal/config"
	"github.com/koki-develop/askai/internal/ui"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

var (
	flagGlobal      bool   // -g, --global
	flagConfigure   bool   // --configure
	flagAPIKey      string // -k, --api-key
	flagModel       string // -m, --model
	flagInteractive bool   // -i, --interactive
)

var rootCmd = &cobra.Command{
	Use:   "askai [flags] [question]",
	Short: "AI is with you",
	Long:  "AI is with you.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagConfigure {
			return configure(cmd, args)
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		uicfg := &ui.Config{
			APIKey:         cfg.APIKey,
			Model:          cfg.Model,
			Interactive:    flagInteractive,
			Messages:       cfg.Messages.OpenAI(),
			SystemMessages: cfg.SystemMessages,
		}

		if cmd.Flag("api-key").Changed {
			uicfg.APIKey = flagAPIKey
		}
		if strings.TrimSpace(uicfg.APIKey) == "" {
			return errors.New("API Key is required")
		}

		if cmd.Flag("model").Changed {
			uicfg.Model = flagModel
		}
		if uicfg.Model == "" {
			uicfg.Model = openai.GPT3Dot5Turbo
		}

		q := ""
		if len(args) > 0 {
			q = strings.Join(args, " ")
		}

		info, err := os.Stdin.Stat()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeCharDevice == 0 {
			b := new(bytes.Buffer)
			if _, err := b.ReadFrom(os.Stdin); err != nil {
				return err
			}
			if q != "" {
				q += "\n\n"
			}
			q += b.String()
		}

		if q != "" {
			uicfg.Question = &q
		}

		ui := ui.New(uicfg)
		if err := ui.Start(); err != nil {
			return err
		}

		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&flagGlobal, "global", "g", false, "configure askai globally (only for --configure)")
	rootCmd.Flags().BoolVar(&flagConfigure, "configure", false, "configure askai")
	rootCmd.Flags().StringVarP(&flagAPIKey, "api-key", "k", "", "the OpenAI API key")
	rootCmd.Flags().StringVarP(&flagModel, "model", "m", openai.GPT3Dot5Turbo, "the chat completion model to use")
	rootCmd.Flags().BoolVarP(&flagInteractive, "interactive", "i", false, "interactive mode")
}
