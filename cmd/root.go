package cmd

import (
	"comp90020-assignment/editor"
	"fmt"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "syncvim [file]",
	Short: "SyncVim is a distributed Vim-like editor",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Usage: syncvim [file] --ip <ip> --port <port>")
			return
		}
		// Your code to start the editor here
		if len(args) >= 1 {
			// Load file
			editor.StartEditor(args[0])
		}
	},
}

var ip string
var port int

func init() {

	rootCmd.PersistentFlags().StringVar(&ip, "ip", "localhost", "IP address for the SyncVim server")
	rootCmd.PersistentFlags().IntVar(&port, "port", 8080, "Port for the SyncVim server")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		// Handle error
	}
}
