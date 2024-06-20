package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	ctLog "github.com/google/certificate-transparency-go/loglist3"
)

func doList(logList *ctLog.LogList) {

	if flags.entryToList != 0 {
		if flags.logToList == "" {
			Abort("Both --log must be specified when using --entry")
		}
		for _, operator := range logList.Operators {
			for _, log := range operator.Logs {
				if strings.HasPrefix(strings.ToUpper(log.URL), strings.ToUpper(flags.logToList)) {
					slog.Debug("Using", "log", log.URL, "operator", operator.Name)
					client := CreateLogClient(log)

					STH, err := client.GetSTH(context.Background())

					if err != nil {
						Abort(err.Error())
					}

					finalEntryIndex := STH.TreeSize

					if finalEntryIndex < flags.entryToList {
						slog.Error("Entry is larger than final entry in log", "entry", flags.entryToList, "finalEntry", finalEntryIndex)
						os.Exit(1)
					}
					entries := GetX509CertLogEntries(client, int64(flags.entryToList), int64(flags.entryToList))
					slog.Debug("Entries", "len()", len(entries))
					for _, entry := range entries {
						var dnsNames []string
						if entry.X509Cert != nil {
							// Not a precert
							dnsNames = entry.X509Cert.DNSNames
						} else if !flags.includePrecert && entry.Precert != nil {
							slog.Warn("Entry has precertificate, use -include-precert to output")
						} else if flags.includePrecert && entry.Precert != nil {
							dnsNames = entry.Precert.TBSCertificate.DNSNames
						}
						for _, dnsName := range dnsNames {
							fmt.Println(dnsName)
						}
					}
					os.Exit(0)
				}
			}

		}
		slog.Error("Failed to find log", "log", flags.logToList, "operator", flags.operatorToList)
		os.Exit(1)
	}

	if flags.logToList != "" {
		fmt.Println("--log does not make sense witout --entry")
		os.Exit(1)
	}

	if flags.operatorToList != "" {
		for _, operator := range logList.Operators {
			if strings.EqualFold(flags.operatorToList, operator.Name) {
				for _, log := range operator.Logs {
					fmt.Println(log.URL)
				}
			}
		}
		os.Exit(0)
	}

	for _, operator := range logList.Operators {
		fmt.Println(operator.Name)
	}
	os.Exit(0)
}
