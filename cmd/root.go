package cmd

import (
	"comp90020-assignment/editor"
	"fmt"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "syncvi [file]",
	Short: "SyncVi is a distributed Vi-like editor",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Usage: syncvi [file] --peers <peers> --me <me> --data-dir <data-dir>")
			return
		}
		if len(args) >= 1 {
			editor.StartEditor(args[0], peers, me, dataDir)
		}
	},
}

var me int32
var peers string
var dataDir string

func init() {
	rootCmd.Flags().StringVarP(&peers, "peers", "p", "localhost:33333", "Comma separated list of peers")
	rootCmd.Flags().Int32VarP(&me, "me", "m", 0, "Index of the current node")
	rootCmd.Flags().StringVarP(&dataDir, "data-dir", "d", ".syncvi", "Directory to store data")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		// Handle error
	}
}
