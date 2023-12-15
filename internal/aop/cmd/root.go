// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"u-control/uc-aom/internal/aop/company"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var verbosity *int
var displayVersionInfo *bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   filepath.Base(os.Args[0]),
	Short: "Create, publish and pull apps.",
	Run: func(cmd *cobra.Command, args []string) {
		if *displayVersionInfo {
			displayVersionThenExit()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	handleDeprecatedSyntax()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(NewPushCommand())
	rootCmd.AddCommand(NewPullCmd())
	rootCmd.AddCommand(NewExportCmd())

	displayVersionInfo = rootCmd.Flags().Bool("version", false, "display version information and exit")
	verbosity = rootCmd.PersistentFlags().CountP("verbose", "v", "explain what is being done, pass multiple times to increase verbosity")
}

func handleDeprecatedSyntax() {
	command, _, err := rootCmd.Find(os.Args[1:])
	if err != nil || !isRootCmd(command) || helpOrVersionInfoRequested(command, os.Args[1:]) {
		return
	}

	log.Warningf("DEPRECATED: No command was provided, push is set automatically for compatibility reasons. Please add the push command to avoid this message.")
	args := append([]string{"push"}, os.Args[1:]...)
	rootCmd.SetArgs(args)
}

func isRootCmd(command *cobra.Command) bool {
	return command.Use == rootCmd.Use
}

func helpOrVersionInfoRequested(command *cobra.Command, args []string) bool {
	return command.Flags().Parse(os.Args[1:]) == pflag.ErrHelp || *displayVersionInfo
}

func displayVersionThenExit() {
	fmt.Fprint(os.Stdout, company.VersionWithCopyrightNotice())
	log.Exit(0)
}

func setLoggingVerbosity() {
	switch *verbosity {
	case 0:
		log.SetLevel(log.WarnLevel)
	case 1:
		log.SetLevel(log.InfoLevel)
	case 2:
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.TraceLevel)
	}
}
