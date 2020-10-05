package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-ping/ping"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/xkortex/vprint"
	"os"
	"runtime"
	"time"
)

type QuietJsonFormatter struct {
	LevelDesc []string
	// TimestampFormat sets the format used for marshaling timestamps.
	TimestampFormat string

	// DisableTimestamp allows disabling automatic timestamps in output
	DisableTimestamp bool

	// DisableHTMLEscape allows disabling html escaping in output
	DisableHTMLEscape bool

	// DataKey allows users to put all the log entry parameters into a nested dictionary at a given key.
	DataKey string

	// FieldMap allows users to customize the names of keys for default fields.
	FieldMap log.FieldMap

	// CallerPrettyfier will do nothing if quiet is selected
	CallerPrettyfier func(*runtime.Frame) (function string, file string)

	// PrettyPrint is off by default for quiet formatter
	PrettyPrint bool
}

//func (f *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
//	timestamp := fmt.Sprintf(entry.Time.Format(f.TimestampFormat))
//	return []byte(fmt.Sprintf("%s %s %s\n", f.LevelDesc[entry.Level], timestamp, entry.Message)), nil
//}

func (f *QuietJsonFormatter) Format(entry *log.Entry) ([]byte, error) {
	data := make(log.Fields, len(entry.Data)+4)
	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/sirupsen/logrus/issues/137
			data[k] = v.Error()
		default:
			data[k] = v
		}
	}

	if f.DataKey != "" {
		newData := make(log.Fields, 4)
		newData[f.DataKey] = data
		data = newData
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	encoder := json.NewEncoder(b)
	encoder.SetEscapeHTML(!f.DisableHTMLEscape)
	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}

	return b.Bytes(), nil
}

var PingCmd = &cobra.Command{
	Use:   "ping",
	Short: "pings a host with ICMP ping",
	Long: `Does what it says on the tin. Ping an endpoint
	`,
	Run: func(cmd *cobra.Command, args []string) {
		quiet, _ := cmd.Flags().GetBool("quiet")
		if quiet {
			log.SetFormatter(&QuietJsonFormatter{})
		} else {
			log.SetFormatter(&log.JSONFormatter{})
		}
		log.SetOutput(os.Stdout)

		vprint.Println(args)
		host := args[0]
		timeout, err := cmd.Flags().GetDuration("timeout")
		if err != nil {
			log.Fatal(err)
		}
		interval, err := cmd.Flags().GetDuration("interval")
		if err != nil {
			log.Fatal(err)
		}
		count, _ := cmd.Flags().GetInt("count")
		if err != nil {
			log.Fatal(err)
		}
		vprint.Println(timeout)
		//if err := cmd.Usage(); err != nil {
		//	log.Fatalf("Error executing root command: %v", err)
		//}
		//log.Fatal("<dbg> silence/usage: ", cmd.SilenceErrors, cmd.SilenceUsage)
		start := time.Now()
		pinger, err := ping.NewPinger(host)
		if err != nil {
			log.Fatal(err)
		}
		pinger.Count = count
		pinger.Timeout = timeout
		pinger.Interval = interval
		err = pinger.Run() // blocks until finished
		if err != nil {
			log.Fatal(err)
		}
		stats := pinger.Statistics() // get send/receive/rtt stats

		if err != nil {
			log.Fatal(err)
		}

		if quiet {
			log.WithFields(log.Fields{
				"addr":  stats.Addr,
				"max":   stats.MaxRtt.Seconds(),
				"loss":  stats.PacketLoss,
				"sent":  stats.PacketsSent,
				"count": count,
			},
			).Info()
		} else {
			log.WithFields(log.Fields{
				"addr":    stats.Addr,
				"elapsed": time.Since(start).Seconds(),
				"max":     stats.MaxRtt.String(),
				"min":     stats.MinRtt.String(),
				"loss":    stats.PacketLoss,
				"sent":    stats.PacketsSent,
				"recv":    stats.PacketsRecv,
			},
			).Info()
		}
		//log.Println(stats)
	},
}

func init() {
	PingCmd.PersistentFlags().IntP("count", "c", 1, "Ping count")
	PingCmd.PersistentFlags().DurationP("interval", "i", time.Duration(5e8), "wait time between each packet send")

	RootCmd.AddCommand(PingCmd)
}
