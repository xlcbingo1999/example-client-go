package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/xlcbingo1999/example-client-go/discoveryclient"
)

var discoveryclientDemoCmd = &cobra.Command{
	Use:   "discoveryclient_demo",
	Short: "Run discoveryclient_demo",
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if err := recover(); err != nil {
				log.Fatalln("Recover err", err)
			}
		}()

		discoveryclient.RunDiscoveryClient()
	},
}

func init() {
	rootCmd.AddCommand(discoveryclientDemoCmd)
}
