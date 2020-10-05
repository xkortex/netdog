package cmd

import (
	"github.com/Wessie/appdirs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/xkortex/vprint"
	"os"
	"path/filepath"
	"time"
)

var (
	cfgFile       string
	developer     string
	defaultCfgDir string
)

const defaultCfgName = "netdog.yml"

var RootCmd = &cobra.Command{
	Use:   "dug",
	Short: "Dug: a better dig",
	Long: `Does what it says on the tin. Bare-bone, no-nonsense DNS lookups 
	`,
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)

		vprint.Printf("root called")
		vprint.Print(args)
		_ = cmd.Help()
		os.Exit(0)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.Fatalf("Error executing root command: %v", err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	defaultCfgDir = appdirs.UserConfigDir("netdog", "", "", false)
	defaultCfgFile := filepath.Join(defaultCfgDir, "config.yml")
	//RootCmd.AddCommand(RootCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// RootCmd.PersistentFlags().String("foo", "", "A help for foo")
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "C",
		defaultCfgFile,
		"config file, based in UserConfigDir")

	RootCmd.PersistentFlags().DurationP("timeout", "t", time.Duration(1e9), "Timeout in seconds")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	RootCmd.PersistentFlags().BoolP("silent", "s", false, "Suppress errors")
	RootCmd.PersistentFlags().BoolP("quiet", "q", false, "Only print essential")
	RootCmd.PersistentFlags().BoolP("stdin", "-", false, "Read from standard in")
	RootCmd.Flags().BoolP("verbose", "v", false, "Verbose tracing (in progress)")
	RootCmd.PersistentFlags().StringVar(&developer, "developer", "Unknown Developer!", "Developer name.")

}

func initConfig() {

}
