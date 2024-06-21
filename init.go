package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	ct "github.com/google/certificate-transparency-go"
)

var DomainToOutput func(string)
var DomainsToOutput func([]string)
var DomainProcessor func([]ct.LogEntry)

type Flags struct {
	// Application loglevel
	appLogLevel string

	// Options for normal operation
	outputFilename  string
	resumeFilename  string
	resume          bool
	lookUpSizeFlag  uint64
	operatorsToSkip arrayFlags
	logsToSkip      arrayFlags
	hasSuffix       string
	onlyOperator    string
	onlyLog         string
	includePrecert  bool

	// List actions
	actionList     bool
	operatorToList string
	logToList      string
	entryToList    uint64
}

var flags Flags

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func init() {

	slog.Debug("Seting up signal handler")
	c := make(chan os.Signal)
	//lint:ignore SA1017 We don't want to buffer signals
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		Abort("Interrupted by user.")
	}()

	// Application loglevel
	flag.StringVar(&flags.appLogLevel, "loglevel", "INFO", "Application loglevel (INFO (default), DEBUG, ERROR, WARN)")

	// Options for normal operation
	var skipResume bool
	flag.BoolVar(&skipResume, "no-resume", false, "Do not use chache (default false)")
	flags.resume = !skipResume
	flag.Uint64Var(&flags.lookUpSizeFlag, "num-entries", 20, "number of entries to query at once")
	flag.StringVar(&flags.outputFilename, "output", "./log-output.log", "Output filename. Use - for stdout. (default ./output.log)")
	flag.StringVar(&flags.resumeFilename, "resume", "./log-resume.json", "Resume filename (default ./log-resume.json)")

	flag.Var(&flags.operatorsToSkip, "skip-operator", "URLs of operators to skip (can be repeated)")
	flag.Var(&flags.logsToSkip, "skip-log", "URLs of logs to skip (can be repeated)")
	flag.StringVar(&flags.onlyOperator, "only-operator", "", "Name of the only operator to process")
	flag.StringVar(&flags.onlyLog, "only-log", "", "URL of the single log to process")
	flag.BoolVar(&flags.includePrecert, "include-precert", false, "Include data from precertificates in output")
	flag.StringVar(&flags.hasSuffix, "required-postfix", "", "Postfixe to require. (will not output domains not matching postfix)")

	// List actions
	flag.BoolVar(&flags.actionList, "list", false, "List something defined by operator/log/entry (default lists operators)")
	flag.StringVar(&flags.operatorToList, "operator", "", "Operator for listing (if only this, will lists logs for this operator)")
	flag.StringVar(&flags.logToList, "log", "", "Log for listing (does not make sense without --entry)")
	flag.Uint64Var(&flags.entryToList, "entry", 0, "Entry to list from --operator's --log")

	slog.Debug("Parsing command line flags")
	flag.Parse()

	if flags.outputFilename == "-" {
		DomainToOutput = DomainToOutputStdout
		DomainsToOutput = DomainsToOutputStdout
	} else {
		DomainToOutput = DomainToOutputFile
		DomainsToOutput = DomainsToOutputFile
	}

	flags.appLogLevel = strings.ToUpper(flags.appLogLevel)

	switch flags.appLogLevel {
	case "DEBUG":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case "INFO":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "WARN":
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case "ERROR":
		slog.SetLogLoggerLevel(slog.LevelError)
	default:
		slog.Error("Log level must be one of INFO, DEBUG, ERROR, WARN")
		os.Exit(1)
	}
}
