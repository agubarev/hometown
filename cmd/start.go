// Copyright © 2019 Andrei Gubarev <agubarev@protonmail.com>

package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"gitlab.com/agubarev/hometown/internal/server"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the main Hometown server.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		log.Fatal(server.StartHometownServer())
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
