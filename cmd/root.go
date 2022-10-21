package cmd

import (
	"fmt"
	"github.com.haa-criticals/watcher/app/grpc"
	"github.com.haa-criticals/watcher/monitor"
	"github.com.haa-criticals/watcher/watcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"time"

	"github.com.haa-criticals/watcher/app"
)

var (
	cfgFile  string
	port     int
	leader   bool
	endpoint string
	url      string

	watch      bool
	provider   string
	baseUrl    string
	token      string
	projectId  int64
	projectRef string
	variables  map[string]string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "watcher",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		w := watcher.New(
			grpc.NewWatchClient(url),
		)
		m := monitor.New(
			monitor.WithHeartBeat(grpc.NewNotifier(), 5*time.Second),
			monitor.WithHealthCheck("https://www.google.com", 5*time.Second, 3),
		)
		a := app.New(w, m, &app.Config{
			Port:           port,
			Leader:         leader,
			LeaderEndpoint: endpoint,
			Address:        url,
		})
		if err := a.Start(); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "c", "config file (default is $HOME/.watcher.yaml)")
	rootCmd.PersistentFlags().StringVar(&provider, "provider", "gitlab", "The provisioner  provider to use, available options are: gitlab")
	rootCmd.PersistentFlags().StringVarP(&baseUrl, "base-url", "b", "", "The base url of the provisioner's provider")
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "The token to use to authenticate with the provisioner's provider")
	rootCmd.PersistentFlags().Int64VarP(&projectId, "project-id", "i", 0, "The project id to use to authenticate with the provisioner's provider")
	rootCmd.PersistentFlags().StringVarP(&projectRef, "project-ref", "r", "main", "The project ref to use with the provisioner's provider")
	rootCmd.PersistentFlags().StringToStringVarP(&variables, "variables", "v", nil, "The variables to use with the provisioner's provider")
	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "The port to use to serve")
	rootCmd.Flags().BoolVarP(&leader, "leader", "l", false, "Determine if this watcher is the leader")
	rootCmd.Flags().StringVarP(&endpoint, "endpoint", "e", "localhost:50051", "The endpoint to use to connect to the leader")
	rootCmd.Flags().StringVarP(&url, "url", "u", "localhost:8080", "The url to use to connect to this watcher")
	_ = rootCmd.MarkFlagRequired("base-url")
	_ = rootCmd.MarkFlagRequired("token")
	_ = rootCmd.MarkFlagRequired("project-id")

	err := viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		log.Printf("Error binding flags: %v", err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".watcher" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".watcher")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		_, _ = fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
