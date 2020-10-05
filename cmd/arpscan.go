/*
Copyright © 2019 MICHAEL McDERMOTT
Apache License 2.0
*/

package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/xkortex/netdog/netdog"
	"github.com/xkortex/vprint"
	"os"
)

var ArpScanCmd = &cobra.Command{
	Use:     "arpscan",
	Short:   "netdog: a better dig",
	Aliases: []string{"as", "arp"},

	Long: `arp scan on an interface 
	`,
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFormatter(&log.JSONFormatter{DisableTimestamp: true})
		log.SetOutput(os.Stdout)

		vprint.Println(args)
		all, _ := cmd.PersistentFlags().GetBool("all")

		timeout, _ := cmd.Flags().GetDuration("timeout")
		delay, _ := cmd.Flags().GetFloat64("delay")
		vprint.Println("Timeout: ", timeout)
		vprint.Println("Delay: ", delay)
		if all {
			arpResults, err := netdog.ScanAllInterfaces(timeout.Seconds(), delay)
			if err != nil {
				log.Fatal(err)
			}
			log.Println(arpResults)
			return
		}
		if len(args) == 0 {
			log.Fatal("Must provide an interface name")
		}
		interfaceName := args[0]

		arpResults, err := netdog.ScanInterface(interfaceName, timeout.Seconds(), delay)
		if err != nil {
			log.Fatal(err)
		}
		vprint.Print(arpResults) // in prog
		return

	},
}

func init() {
	RootCmd.AddCommand(ArpScanCmd)
	ArpScanCmd.PersistentFlags().BoolP("all", "a", false, "Scan all interfaces")
	ArpScanCmd.PersistentFlags().Float64P("delay", "d", 0.0, "Delay between sends in seconds")

}
