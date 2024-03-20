package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/xlcbingo1999/example-client-go/dynamicclient"
)

var dynamicclientDemoCmd = &cobra.Command{
	Use:   "dynamicclient_demo",
	Short: "Run dynamicclient_demo",
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if err := recover(); err != nil {
				log.Fatalln("Recover err", err)
			}
		}()

		dynamicclient.RunDynamicClient()
	},
}

func init() {
	rootCmd.AddCommand(dynamicclientDemoCmd)
}
