package cmd

import (
	"comp90020-assignment/editor"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strconv"
)

var rootCmd = &cobra.Command{
	Use:   "syncvi [file]",
	Short: "SyncVi is a distributed Vi-like editor",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		logFile, err := os.OpenFile("syncvi.log"+strconv.FormatInt(int64(me), 10), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Error opening log file: %v", err)
		}
		log.SetOutput(logFile)
		if len(args) < 1 {
			fmt.Println("Usage: syncvi [file] --peers <peers> --me <me> --data-dir <data-dir>")
			return
		}
		if len(args) >= 1 {
			editor.StartEditor(args[0], raftPeers, nodes, me, dataDir)
		}
	},
}

var me int32
var raftPeers string
var nodes string
var dataDir string

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/"

}

func init() {
	rootCmd.Flags().StringVarP(&raftPeers, "peers", "p", "localhost:22222", "Raft peers")
	rootCmd.Flags().StringVarP(&nodes, "nodes", "n", "localhost:23333", "Nodes")
	rootCmd.Flags().Int32VarP(&me, "me", "m", 0, "Index of the current node")
	rootCmd.Flags().StringVarP(&dataDir, "data-dir", "d", userHomeDir()+"/.syncvi", "Directory to store data")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		// Handle error
	}
}
