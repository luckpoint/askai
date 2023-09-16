package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/koki-develop/askai/internal/ui"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

/*
 * TODO: refactor
 */

var (
	cfg config

	flagGlobal      bool   // -g, --global
	flagAPIKey      string // -k, --api-key
	flagModel       string // -m, --model
	flagInteractive bool   // -i, --interactive
)

type config struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure askai",
	Long:  "Configure askai.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flag("api-key").Changed && !cmd.Flag("model").Changed {
			fmt.Print("OpenAI API Key: ")
			key, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return err
			}
			if len(key) != 0 {
				cfg.APIKey = string(key)
			}

			fmt.Print("\nChat Completion Model: ")
			var m string
			fmt.Scanln(&m)
			if m != "" {
				cfg.Model = m
			}
		}

		if cmd.Flag("api-key").Changed {
			viper.Set("api_key", flagAPIKey)
		} else if cfg.APIKey != "" {
			viper.Set("api_key", cfg.APIKey)
		}
		if cmd.Flag("model").Changed {
			viper.Set("model", flagModel)
		} else if cfg.Model != "" {
			viper.Set("model", cfg.Model)
		}

		if flagGlobal {
			h, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			p := filepath.Join(h, ".askai")
			viper.SetConfigFile(p)
		} else {
			viper.SetConfigFile(".askai")
		}
		if err := viper.WriteConfig(); err != nil {
			return err
		}

		fmt.Println("Configured.")
		return nil
	},
}

var rootCmd = &cobra.Command{
	Use:   "askai [flags] [question]",
	Short: "AI is with you",
	Long:  "AI is with you.",
	RunE: func(cmd *cobra.Command, args []string) error {
		uicfg := &ui.Config{
			APIKey:      cfg.APIKey,
			Model:       cfg.Model,
			Interactive: flagInteractive,
		}

		if cmd.Flag("api-key").Changed {
			uicfg.APIKey = flagAPIKey
		}
		if uicfg.APIKey == "" {
			return fmt.Errorf("OpenAI API Key is required")
		}

		if cmd.Flag("model").Changed {
			uicfg.Model = flagModel
		}
		if uicfg.Model == "" {
			uicfg.Model = openai.GPT3Dot5Turbo
		}

		q := strings.Join(args, " ")
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
	rootCmd.Flags().StringVarP(&flagAPIKey, "api-key", "k", "", "the OpenAI API key")
	rootCmd.Flags().StringVarP(&flagModel, "model", "m", openai.GPT3Dot5Turbo, "the chat completion model to use")
	rootCmd.Flags().BoolVarP(&flagInteractive, "interactive", "i", false, "interactive mode")

	rootCmd.AddCommand(configureCmd)
	configureCmd.Flags().BoolVarP(&flagGlobal, "global", "g", false, "configure askai globally")
	configureCmd.Flags().StringVarP(&flagAPIKey, "api-key", "k", "", "the OpenAI API key")
	configureCmd.Flags().StringVarP(&flagModel, "model", "m", "", "the chat completion model to use")

	cobra.OnInitialize(func() {
		viper.SetConfigName(".askai")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
		_ = viper.ReadInConfig()
		_ = viper.Unmarshal(&cfg)
	})
}
