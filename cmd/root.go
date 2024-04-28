package cmd

import (
	"comp90020-assignment/editor"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"strconv"
)

var rootCmd = &cobra.Command{
	Use:   "syncvi",
	Short: "SyncVi is a distributed Vi-like editor",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the editor",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		if showLog {
			if os.MkdirAll(dataDir, os.ModePerm) != nil {
				fmt.Errorf("Error creating directory: %v", dataDir)
			}
			logFile, err := os.OpenFile(dataDir+"/syncvi.log"+strconv.FormatInt(int64(me), 10), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Fatalf("Error opening log file: %v", err)
			}
			log.SetOutput(logFile)
		} else {
			log.SetOutput(io.Discard)
		}
		if len(args) < 1 {
			fmt.Println("Usage: syncvi [file] --peers <peers> --me <me> --data-dir <data-dir>")
			return
		}
		if len(args) >= 1 {
			editor.StartEditor(args[0], raftPeers, nodes, me, dataDir)
		}
	},
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean the editor before starting a new document",
	Run: func(cmd *cobra.Command, args []string) {
		editor.Clean(dataDir)
	},
}

var me int32
var raftPeers string
var nodes string
var dataDir string
var showLog bool

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/"
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(cleanCmd)

	startCmd.Flags().StringVarP(&raftPeers, "peers", "p", "localhost:22222", "Raft peers, format: <ip>:<port>,<ip>:<port>,...")
	startCmd.Flags().StringVarP(&nodes, "nodes", "n", "localhost:23333", "Nodes, format: <ip>:<port>,<ip>:<port>,...")
	startCmd.Flags().Int32VarP(&me, "me", "m", 0, "Index of the current node, start from 0")
	startCmd.Flags().StringVarP(&dataDir, "data-dir", "d", userHomeDir()+".syncvi", "Directory to store data")
	startCmd.Flags().BoolVarP(&showLog, "show-log", "l", false, "Show log, log will print to data-dir/syncvi.log<me>")

	cleanCmd.Flags().StringVarP(&dataDir, "data-dir", "d", userHomeDir()+".syncvi", "Directory to store data")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
