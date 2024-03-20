package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/xlcbingo1999/example-client-go/informer"
)

var informerDemoCmd = &cobra.Command{
	Use:   "informer_demo",
	Short: "Run informer_demo",
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if err := recover(); err != nil {
				log.Fatalln("Recover err", err)
			}
		}()

		informer.RunInformer()
	},
}

func init() {
	rootCmd.AddCommand(informerDemoCmd)
}
