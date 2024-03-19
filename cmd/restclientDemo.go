package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/xlcbingo1999/example-client-go/restclient"
)

var restclientDemoCmd = &cobra.Command{
	Use:   "restclient_demo",
	Short: "Run restclient_demo",
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if err := recover(); err != nil {
				log.Fatalln("Recover err", err)
			}
		}()

		restclient.RunRestClient()
	},
}

func init() {
	rootCmd.AddCommand(restclientDemoCmd)
}
