package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/spf13/cobra"
	"io"
	log "github.com/sirupsen/logrus"
	"math/big"
	"net/http"
	"os"
	"sync"
	"time"

	quic "github.com/lucas-clemente/quic-go"
)

const default_addr = "localhost:4242"

const default_message = "foobar"

// We start a server echoing data on the first stream the client opens,
// then connect with a client, send the message, and wait for its receipt.
func selfMain() {
	go func() { log.Fatal(echoServer(default_addr)) }()

	err := clientMain(default_addr, default_message, 1)
	if err != nil {
		panic(err)
	}
}

// Start a server that echos all data on the first stream opened by the client
func echoServer(addr string) error {
	listener, err := quic.ListenAddr(addr, generateTLSConfig(), nil)
	if err != nil {
		return err
	}
	sess, err := listener.Accept(context.Background())
	if err != nil {
		return err
	}
	stream, err := sess.AcceptStream(context.Background())
	if err != nil {
		panic(err)
	}
	// Echo through the loggingWriter
	_, err = io.Copy(loggingWriter{stream}, stream)
	return err
}

func serveSession(sess quic.Session, wg *sync.WaitGroup) {
	log.Infof("New session\n")
	stream, err := sess.AcceptStream(context.Background())
	if err != nil {
		panic(err)
	}
	// Echo through the loggingWriter
	_, err = io.Copy(loggingWriter{stream}, stream)
	if err != nil {
		log.Error(err)
	}
	wg.Done()
}

func listenAndServe(addr string) error {
	log.Info("Starting listener")
	listener, err := quic.ListenAddr(addr, generateTLSConfig(), nil)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	for {
		sess, err := listener.Accept(context.Background())
		if err != nil {
			return err
		}
		wg.Add(1)
		go serveSession(sess, &wg)
	}
	wg.Wait()

	return nil
}

func pingOnce(stream quic.Stream, message string ) (buf []byte, err error) {
	_, err = stream.Write([]byte(message))
	if err != nil {
		return nil, err
	}

	buf = make([]byte, len(message))
	_, err = io.ReadFull(stream, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func clientMain(addr string, message string, count int) error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}
	session, err := quic.DialAddr(addr, tlsConf, nil)
	if err != nil {
		return err
	}

	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		return err
	}
	totalElapsed := time.Duration(0)

	pingOnce(stream, message) // eat any startup timing costs
	for i := 0; i < count; i++ {
		fmt.Printf("Client: Sending '%s'\n", message)
		start := time.Now()

		buf, err := pingOnce(stream, message)

		if err != nil {
			return err
		}
		elapsed := time.Since(start)
		totalElapsed += elapsed

		fmt.Printf("Client: Got '%s'\n", buf)
	}
	timePer := float64(totalElapsed.Microseconds()) / float64(count)

	fmt.Printf("%8d: elapsed: %7.2f µs \n", count, timePer, )


	return nil
}

func client() error {

	quiet := flag.Bool("q", false, "don't print the data")
	keyLogFile := flag.String("keylog", "", "key log file")
	insecure := flag.Bool("insecure", false, "skip certificate verification")
	flag.Parse()
	urls := flag.Args()



	var keyLog io.Writer
	if len(*keyLogFile) > 0 {
		f, err := os.Create(*keyLogFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		keyLog = f
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatal(err)
	}
	//testdata.AddRootCA(pool)

	var qconf quic.Config
	roundTripper := &http3.RoundTripper{
		TLSClientConfig: &tls.Config{
			RootCAs:            pool,
			InsecureSkipVerify: *insecure,
			KeyLogWriter:       keyLog,
		},
		QuicConfig: &qconf,
	}
	defer roundTripper.Close()
	hclient := &http.Client{
		Transport: roundTripper,
	}

	var wg sync.WaitGroup
	wg.Add(len(urls))
	for _, addr := range urls {
		log.Infof("GET %s", addr)
		go func(addr string) {
			rsp, err := hclient.Get(addr)
			if err != nil {
				log.Fatal(err)
			}
			log.Infof("Got response for %s: %#v", addr, rsp)

			body := &bytes.Buffer{}
			_, err = io.Copy(body, rsp.Body)
			if err != nil {
				log.Fatal(err)
			}
			if *quiet {
				log.Infof("Request Body: %d bytes", body.Len())
			} else {
				log.Infof("Request Body:")
				log.Infof("%s", body.Bytes())
			}
			wg.Done()
		}(addr)
	}
	wg.Wait()
	return nil
}

// A wrapper for io.Writer that also logs the message.
type loggingWriter struct{ io.Writer }

func (w loggingWriter) Write(b []byte) (int, error) {
	fmt.Printf("Server: Got '%s'\n", string(b))
	return w.Writer.Write(b)
}

// Setup a bare-bones TLS config for the server
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}
}

var EchoCmd = &cobra.Command{
	Use:   "echo",
	Short: "pings a host with ICMP ping",
	Long: `Does what it says on the tin. Ping an endpoint
	`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("echo root")
	},
}
var EchoServeCmd = &cobra.Command{
	Use:   "serve",
	Aliases: []string{"s"},
	Short: "serves as an echo host",
	Long: `starts a server which listens and responds with an echo
	`,
	Run: func(cmd *cobra.Command, args []string) {
		addr, _ := cmd.Flags().GetString("addr")
		log.Infof("serving: %s\n", addr)
		log.Fatal(listenAndServe(addr))
		//selfMain()
	},
}
var EchoClientCmd = &cobra.Command{
	Use:   "client",
	Aliases: []string{"c"},
	Short: "serves as an echo client",
	Long: `starts a client which talks to an echo server
	`,
	Run: func(cmd *cobra.Command, args []string) {
		count, _ := cmd.Flags().GetInt("count")
		addr, _ := cmd.Flags().GetString("addr")
		msg, _ := cmd.Flags().GetString("msg")

		clientMain(addr, msg, count)
	},
}
var EchoTestCmd = &cobra.Command{
	Use:   "test",
	Aliases: []string{"t"},
	Short: "run internal test",
	Long: `starts a server and a client
	`,
	Run: func(cmd *cobra.Command, args []string) {
		selfMain()
	},
}

func init() {
	EchoCmd.PersistentFlags().DurationP("interval", "i", time.Duration(5e8), "wait time between each packet send")

	EchoCmd.PersistentFlags().StringP("addr", "a", default_addr, "Address to attach to")
	EchoClientCmd.PersistentFlags().StringP("msg", "m", default_message, "Message to send")
	EchoClientCmd.PersistentFlags().IntP("count", "c", 1, "Ping count")

	EchoCmd.AddCommand(EchoServeCmd)
	EchoCmd.AddCommand(EchoClientCmd)

	EchoCmd.AddCommand(EchoTestCmd)

	RootCmd.AddCommand(EchoCmd)
	}