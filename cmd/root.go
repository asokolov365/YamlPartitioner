// Copyright 2023-2024 Andrew Sokolov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/asokolov365/YamlPartitioner/app"
	"github.com/asokolov365/YamlPartitioner/version"
	"github.com/asokolov365/snakecharmer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "yp",
	Short: "*yp* partitions input YAML file(s) using the rendezvous consistent hashing algorithm.",
	// Long:  `A longer description that spans multiple lines and likely contains`,

	// examples and usage of using your application. For example:
	Example: `# This will partition the input file to 5 shards
# each rule under "groups.*.rules" will be written to 2 shards,
# the result will be stored in /tmp/node3/recording-rules.yml
> yp --src="./recording-rules.yml" \
  --split-at="groups.*.rules" \
  --shards-count=5 \
  --replication=2 \
  --shard-id=3 \
  --dst="/tmp" \
  --shard-basename="node"`,

	// Disable automatic printing of usage information whenever an error
	// occurs. Many errors are not the result of a bad command invocation,
	// e.g. attempting to start a node on an in-use port, and printing the
	// usage information in these cases obscures the cause of the error.
	// Commands should manually print usage information when the error is,
	// in fact, a result of a bad invocation, e.g. too many arguments.
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		// cmd.DebugFlags()
		// This fills out the Config struct.
		if err = charmer.UnmarshalExact(); err != nil {
			if errUsage := cmd.Usage(); errUsage != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", errUsage.Error())
			}
			return err
		}

		if err = app.Init(); err != nil {
			return err
		}

		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// Create a context that cancels when OS signals come in.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer stop()

		// Testing signal
		// p, err := os.FindProcess(os.Getpid())
		// if err != nil {
		// 	return err
		// }
		// On a Unix-like system, pressing Ctrl+C on a keyboard sends a
		// SIGINT signal to the process of the program in execution.
		//
		// This example simulates that by sending a SIGINT signal to itself.
		// go func() {
		// 	time.Sleep(10 * time.Millisecond)
		// 	if err := p.Signal(os.Interrupt); err != nil {
		// 		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		// 	}
		// }()

		if err := app.Run(ctx, verbose); err != nil {
			return err
		}

		return nil
	},

	Version: fmt.Sprintf("Version: %s, GitCommit: %s\n", version.Version, version.GitCommit),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

var (
	vpr     *viper.Viper
	charmer *snakecharmer.SnakeCharmer
	verbose bool
)

func init() {
	var err error
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print report of partitioning of each input YAML file.")
	rootCmd.SetVersionTemplate(`{{printf "YamlPartitioner *yp* %s" .Version}}`)
	vpr = viper.New()
	// This inits app.MainConfig with default values
	app.InitConfig()

	charmer, err = snakecharmer.NewSnakeCharmer(
		snakecharmer.WithCobraCommand(rootCmd),
		snakecharmer.WithViper(vpr),
		snakecharmer.WithResultStruct(app.MainConfig),
	)
	if err != nil {
		panic(fmt.Sprintf("error init SnakeCharmer: %s", err.Error()))
	}

	// This adds Flags automatically generated from the app.MainConfig struct
	charmer.AddFlags()

	// See config.go for the complete list of the flags
	// rootCmd.MarkPersistentFlagRequired("src")
	if err = rootCmd.MarkPersistentFlagRequired("split-at"); err != nil {
		panic(err)
	}
	if err = rootCmd.MarkPersistentFlagRequired("shards-number"); err != nil {
		panic(err)
	}
}
