package main

import (
	"os"

	"github.com/YunDingLab/fix_log4j2/internal/config"
	"github.com/YunDingLab/fix_log4j2/internal/fix"
	"github.com/YunDingLab/fix_log4j2/internal/global"
	"github.com/YunDingLab/fix_log4j2/internal/logs"
	"github.com/YunDingLab/fix_log4j2/version"
	"github.com/spf13/cobra"
)

var (
	configfile string
	showVer    bool

	rootCmd = &cobra.Command{
		Use:   "puppy",
		Short: "Process Lifecycle Manager",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if showVer {
				version.Fprint(os.Stdout)
				os.Exit(0)
			}
			_, err := config.LoadConfig(configfile)
			if err != nil {
				return err
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// logs.Infof("[main] config: %+v", opt)
			cluster, err := fix.NewCluster()
			if err != nil {
				return
			}
			logs.Infof("[main] got kube cluster succ.")
			err = cluster.RunCheck()
			if err != nil {
				return
			}
			logs.Infof("[main] done succ and exit.")
		},
	}
)

func main() {
	rootCmd.PersistentFlags().StringVarP(&configfile, "config", "c", "./internal/config/example.yaml", "config file path.")
	rootCmd.PersistentFlags().BoolVarP(&global.Debug, "debug", "D", false, "debug")
	rootCmd.Flags().BoolVarP(&showVer, "version", "v", false, "print version")
	rootCmd.Execute()
}
