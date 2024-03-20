package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/xlcbingo1999/example-client-go/clientset"
)

var clientsetDemoCmd = &cobra.Command{
	Use:   "clientset_demo",
	Short: "Run clientset_demo",
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if err := recover(); err != nil {
				log.Fatalln("Recover err", err)
			}
		}()

		clientset.RunClientSet()
	},
}

func init() {
	clientsetDemoCmd.Flags().StringVarP(&clientset.Operate, "operate", "", "create", "operate type : create or clean or list")
	rootCmd.AddCommand(clientsetDemoCmd)
}
