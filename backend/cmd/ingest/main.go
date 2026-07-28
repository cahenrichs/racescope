package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/clint/f1/backend/internal/openf1"
)

type options struct {
	season     int
	meetingKey int
}

func main() {
	options, err := parseOptions(os.Args[1:], os.Stderr, time.Now().UTC().Year())
	if err != nil {
		log.Fatal(err)
	}

	// The remaining Task 8 steps will pass this validated target to the importer.
	log.Printf("weekend import target selected: season=%d meeting=%d", options.season, options.meetingKey)
}

func parseOptions(args []string, output io.Writer, currentYear int) (options, error) {
	flags := flag.NewFlagSet("ingest", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.Usage = func() {
		fmt.Fprintln(output, "Usage: ingest --season YEAR --meeting OPENF1_MEETING_KEY")
		flags.PrintDefaults()
	}

	var parsed options
	flags.IntVar(&parsed.season, "season", 0, "season year to import")
	flags.IntVar(&parsed.meetingKey, "meeting", 0, "OpenF1 meeting key to import")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if parsed.season < openf1.FirstSupportedYear || parsed.season > currentYear {
		return options{}, fmt.Errorf("season must be between %d and %d", openf1.FirstSupportedYear, currentYear)
	}
	if parsed.meetingKey <= 0 {
		return options{}, errors.New("meeting must be a positive OpenF1 meeting key")
	}

	return parsed, nil
}
