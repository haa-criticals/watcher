package cmd

import (
	"context"
	"fmt"
	"github.com.haa-criticals/watcher/app/grpc"
	"github.com.haa-criticals/watcher/monitor"
	"github.com.haa-criticals/watcher/provisioner"
	"github.com.haa-criticals/watcher/watcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"time"

	"github.com.haa-criticals/watcher/app"
)

var (
	cfgFile             string
	address             string
	peers               []string
	heartBeatInterval   uint64
	healthCheckInterval uint64
	watch               bool
	baseUrl             string
	token               string
	projectId           int64
	projectRef          string
	variables           map[string]string
)

type providerConsole struct {
}

func (p *providerConsole) Create(_ context.Context) error {
	log.Println("Provider called Create")
	return nil
}

func (p *providerConsole) Destroy(_ context.Context) error {
	log.Println("Provider called Destroy")
	return nil
}

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
		w := watcher.New(grpc.NewWatchClient(), watcher.Config{
			Address:                address,
			HeartBeatCheckInterval: time.Duration(heartBeatInterval+1) * time.Second,
			MaxDelayForElection:    5000,
		})
		m := monitor.New(
			monitor.WithHeartBeat(grpc.NewWatchClient(), time.Duration(heartBeatInterval)*time.Second),
			monitor.WithHealthCheck("https://www.google.com", time.Duration(healthCheckInterval)*time.Second, 3),
		)

		a := app.New(w, m, provisioner.WithProvider(&providerConsole{}), &app.Config{
			Address: address,
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
	rootCmd.PersistentFlags().StringVarP(&baseUrl, "base-url", "b", "", "The base url of the provisioner's provider")
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "The token to use to authenticate with the provisioner's provider")
	rootCmd.PersistentFlags().Int64VarP(&projectId, "project-id", "i", 0, "The project id to use to authenticate with the provisioner's provider")
	rootCmd.PersistentFlags().StringVarP(&projectRef, "project-ref", "r", "main", "The project ref to use with the provisioner's provider")
	rootCmd.PersistentFlags().StringToStringVarP(&variables, "variables", "v", nil, "The variables to use with the provisioner's provider")
	rootCmd.Flags().StringVarP(&address, "address", "a", ":8080", "The bind address this watcher")
	rootCmd.Flags().StringSliceVarP(&peers, "peers", "p", nil, "The peers to connect to")
	rootCmd.Flags().Uint64VarP(&heartBeatInterval, "heartbeat-interval", "hb", 5, "The interval to send heartbeats in seconds")
	rootCmd.Flags().Uint64VarP(&healthCheckInterval, "healthcheck-interval", "hc", 5, "The interval to send healthchecks in seconds")
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
