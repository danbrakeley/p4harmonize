package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/danbrakeley/frog"
	"github.com/proletariatgames/p4harmonize/internal/buildvar"
)

func PrintUsage() {
	version := "<local build>"
	if len(buildvar.Version) > 0 {
		version = buildvar.Version
	}
	buildTime := "<not set>"
	if len(buildvar.BuildTime) > 0 {
		buildTime = buildvar.BuildTime
	}
	url := "https://github.com/proletariatgames/p4harmonize"
	if len(buildvar.ReleaseURL) > 0 {
		url = buildvar.ReleaseURL
	}

	fmt.Fprintf(os.Stderr,
		strings.Join([]string{
			"",
			"p4harmonize %s, created %s",
			"%s",
			"",
			"Usage:",
			"\tp4harmonize [--config PATH]",
			"\tp4harmonize --version",
			"\tp4harmonize --help",
			"Options:",
			"\t-c, --config PATH     Config file location (default: 'config.toml')",
			"\t-v, --version         Print just the version number (to stdout)",
			"\t-h, --help            Print this message (to stderr)",
			"",
			"Config files must be in TOML format. See the README for an example.",
			"",
		}, "\n"), version, buildTime, url,
	)
}

func main() {
	status := mainExit()
	if status != 0 {
		// From os/proc.go: "For portability, the status code should be in the range [0, 125]."
		if status < 0 || status > 125 {
			status = 125
		}
		os.Exit(status)
	}
}

func mainExit() int {
	flag.Usage = PrintUsage

	var cfgPath string
	var showVersion bool
	var showHelp bool
	flag.StringVar(&cfgPath, "c", "config.toml", "config file location")
	flag.StringVar(&cfgPath, "config", "config.toml", "config file location")
	flag.BoolVar(&showVersion, "v", false, "show version info")
	flag.BoolVar(&showVersion, "version", false, "show version info")
	flag.BoolVar(&showHelp, "h", false, "show version info")
	flag.BoolVar(&showHelp, "help", false, "show version info")
	flag.Parse()

	if showVersion {
		if len(buildvar.Version) == 0 {
			fmt.Printf("unknown\n")
			return 1
		}
		fmt.Printf("%s\n", strings.TrimPrefix(buildvar.Version, "v"))
		return 0
	}

	if showHelp {
		flag.Usage()
		return 0
	}

	if len(flag.Args()) > 0 {
		fmt.Printf("unrecognized arguments: %v\n", strings.Join(flag.Args(), " "))
		flag.Usage()
		return 1
	}

	start := time.Now()
	log := MakeLogger(frog.New(frog.Auto, frog.HideLevel, frog.MessageOnRight, frog.FieldIndent10), "")
	defer func() {
		dur := time.Now().Sub(start)
		log.Info("Running Time: %v", dur)
		log.Logger.SetMinLevel(frog.Info)
		log.Close()
	}()

	cfg, err := loadConfigFromFile(cfgPath)
	if err != nil {
		log.Error("Failed to load config: %v", err)
		return 1
	}

	log.Info("Config loaded from %s", cfg.Filename())

	err = Harmonize(log, cfg)
	if err != nil {
		log.Error("%v", err)
		return 2
	}

	return 0
}
