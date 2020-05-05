package cmd

import (
	"github.com/Wessie/appdirs"
	"github.com/spf13/cobra"
	"github.com/xkortex/netdog/netdog"
	"github.com/xkortex/vprint"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
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
		host := args[0]
		ns, _ := cmd.PersistentFlags().GetString("namespace")
		timeout, _ := cmd.PersistentFlags().GetFloat64("timeout")
		vprint.Printf(ns)
		//if err := cmd.Usage(); err != nil {
		//	log.Fatalf("Error executing root command: %v", err)
		//}
		//log.Fatal("<dbg> silence/usage: ", cmd.SilenceErrors, cmd.SilenceUsage)
		addrs, err := netdog.TimeoutLookupHost(host, timeout)

		if err != nil {
			log.Fatal(err)
		}
		log.WithFields(log.Fields{
			"addrs": addrs,
			"host": host}).Info()
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
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c",
		defaultCfgFile,
		"config file, based in UserConfigDir", )

	RootCmd.PersistentFlags().Float64P("timeout", "t", 0.1, "Timeout in seconds")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	RootCmd.PersistentFlags().BoolP("silent", "s", false, "Suppress errors")
	RootCmd.PersistentFlags().BoolP("stdin", "-", false, "Read from standard in")
	RootCmd.Flags().BoolP("verbose", "v", false, "Verbose tracing (in progress)")
	RootCmd.PersistentFlags().StringVar(&developer, "developer", "Unknown Developer!", "Developer name.")

}

func initConfig() {

}
