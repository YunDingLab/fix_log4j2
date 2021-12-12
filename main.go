package main

import (
	"github.com/YunDingLab/fix_log4j2/internal/config"
	"github.com/YunDingLab/fix_log4j2/internal/fix"
	"github.com/YunDingLab/fix_log4j2/internal/global"
	"github.com/YunDingLab/fix_log4j2/internal/logs"
	"github.com/spf13/cobra"
)

var (
	configfile string

	rootCmd = &cobra.Command{
		Use:   "puppy",
		Short: "Process Lifecycle Manager",
		PreRunE: func(cmd *cobra.Command, args []string) error {
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
	rootCmd.Execute()
}
