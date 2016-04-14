package main

import (
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/pantheon-systems/journal-2-logstash/journal_2_logstash"
)

type options struct {
	Debug     bool   `short:"d" long:"debug" description:"enable debug output" default:"false" env:"JOURNAL2LOGSTASH_DEBUG"`
	Socket    string `short:"s" long:"socket" description:"Path to systemd-journal-gatewayd unix socket" env:"JOURNAL2LOGSTASH_SOCKET" required:"true"`
	URL       string `short:"u" long:"url" description:"URL (host:port) to Logstash TLS server" env:"JOURNAL2LOGSTASH_URL" required:"true"`
	Key       string `short:"k" long:"key" description:"Path to client TLS key to use when contacting Logstash server" env:"JOURNAL2LOGSTASH_TLS_KEY" required:"true"`
	Cert      string `short:"c" long:"cert" description:"Path to client TLS cert to use when contacting Logstash server" env:"JOURNAL2LOGSTASH_TLS_CERT" required:"true"`
	Ca        string `short:"a" long:"ca" description:"Path to CA bundle for authenticating Logstash TLS server" env:"JOURNAL2LOGSTASH_TLS_CA" required:"true"`
	StateFile string `short:"t" long:"state" description:"Path to file to save state between invocations" env:"JOURNAL2LOGSTASH_STATE_FILE" required:"true"`
}

func parseArgs(args []string) (*options, error) {
	opts := &options{}
	parser := flags.NewParser(opts, flags.PassDoubleDash|flags.HelpFlag)
	_, err := parser.ParseArgs(args)
	if err != nil {
		return nil, err
	}
	return opts, nil
}

func main() {
	opts, err := parseArgs(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	cfg := journal_2_logstash.JournalShipperConfig{
		Debug:     opts.Debug,
		StateFile: opts.StateFile,
		Socket:    opts.Socket,
		URL:       opts.URL,
		Key:       opts.Key,
		Cert:      opts.Cert,
		Ca:        opts.Ca,
	}
	shipper, err := journal_2_logstash.NewShipper(cfg)
	if err != nil {
		log.Fatal("Exiting:", err)
	}

	err = shipper.Run()
	if err != nil {
		log.Fatal("Exiting:", err)
	}
}
