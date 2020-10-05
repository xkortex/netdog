package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/xkortex/netdog/netdog"
	"github.com/xkortex/vprint"
	"os"
	"time"
)

var DigCmd = &cobra.Command{
	Use:   "dig",
	Short: "netdog: like dig, but fluffier",
	Long: `Does what it says on the tin. Bare-bone, no-nonsense DNS lookups 
	`,
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)

		vprint.Printf("root called")
		vprint.Println(args)
		host := args[0]
		timeout, _ := cmd.Flags().GetDuration("timeout")
		vprint.Println(timeout)
		//if err := cmd.Usage(); err != nil {
		//	log.Fatalf("Error executing root command: %v", err)
		//}
		//log.Fatal("<dbg> silence/usage: ", cmd.SilenceErrors, cmd.SilenceUsage)
		start := time.Now()
		addrs, err := netdog.TimeoutLookupHost(host, timeout.Seconds())

		if err != nil {
			log.Fatal(err)
		}
		log.WithFields(log.Fields{
			"addrs":   addrs,
			"host":    host,
			"elapsed": time.Since(start).Seconds()},
		).Info()
	},
}

func init() {
	RootCmd.AddCommand(DigCmd)
}
