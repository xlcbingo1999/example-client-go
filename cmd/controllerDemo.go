package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/xlcbingo1999/example-client-go/controller"
)

var controllerDemoCmd = &cobra.Command{
	Use:   "controller_demo",
	Short: "Run controller_demo",
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if err := recover(); err != nil {
				log.Fatalln("Recover err", err)
			}
		}()

		controller.RunController()
	},
}

func init() {
	rootCmd.AddCommand(controllerDemoCmd)
}
