package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use:   "mpa-service",
	Short: "Multi-Protocol Agent for IoT data integration",
	Long: `MPA (Multi-Protocol Agent) is a service that receives device payloads 
from multiple IoT protocols via HTTP and forwards them to an MQTT broker.
Designed with extensible architecture for handling various IoT data sources 
with unified interface and message format.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is configs/config.yaml)")
}