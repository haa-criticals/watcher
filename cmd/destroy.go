package cmd

import (
	"fmt"
	"github.com.haa-criticals/watcher/provisioner"
	"log"

	"github.com/spf13/cobra"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("destroying infrastructure")
		m := provisioner.New(&provisioner.Config{
			BaseUrl:   baseUrl,
			Token:     token,
			ProjectId: projectId,
			Ref:       projectRef,
			Variables: variables,
			Watch:     watch,
		})
		if err := m.Destroy(cmd.Context()); err != nil {
			log.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// destroyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// destroyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	destroyCmd.Flags().BoolVar(&watch, "watch", false, "watch for progress")
}
