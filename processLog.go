package main

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	ct "github.com/google/certificate-transparency-go"
	ctLog "github.com/google/certificate-transparency-go/loglist3"
)

var totalProcessed uint64
var currentlyProcessing bool

func ProcessDomainsFromEntries(entries []ct.LogEntry) {
	for _, entry := range entries {
		if entry.X509Cert != nil {
			for _, dnsName := range entry.X509Cert.DNSNames {
				if flags.hasSuffix != "" {
					if strings.HasSuffix(dnsName, flags.hasSuffix) {
						DomainToOutput(dnsName)
					}
				} else if !strings.HasSuffix(dnsName, "flowers-to-the-world.com") {
					DomainToOutput(dnsName)
				}
			}
		}
	}
}

func ProcessDomainsFromEntriesIncludingPrecerts(entries []ct.LogEntry) {
	ProcessDomainsFromEntries(entries)
	for _, entry := range entries {
		if entry.Precert != nil {
			for _, dnsName := range entry.Precert.TBSCertificate.DNSNames {
				if flags.hasSuffix != "" {
					if strings.HasSuffix(dnsName, flags.hasSuffix) {
						DomainToOutput(dnsName)
					}
				} else if !strings.HasSuffix(dnsName, "flowers-to-the-world.com") {
					DomainToOutput(dnsName)
				}
			}
		}
	}
}

func ProcessLog(wg *sync.WaitGroup, log *ctLog.Log, operator *ctLog.Operator) {
	defer wg.Done()

	slog.Debug("Querying log ", "url", log.URL, "operator", operator.Name)
	client := CreateLogClient(log)

	STH, err := client.GetSTH(context.Background())

	if err != nil {
		Abort(err.Error())
	}

	finalEntryIndex := STH.TreeSize

	currentIndexInt, _ := mep.Get(log.URL)
	lookupSize := flags.lookUpSizeFlag
	currentIndex := uint64(currentIndexInt)
	lastPercent := uint64(999)
	for currentIndex < finalEntryIndex {
		if currentIndex+lookupSize > finalEntryIndex {
			lookupSize = finalEntryIndex - currentIndex
		}
		prcnt := currentIndex / finalEntryIndex
		if lastPercent == 999 || prcnt > lastPercent {
			slog.Info("Progress", "log", log.URL, "prcnt", prcnt)
			lastPercent = prcnt
		}
		slog.Debug("Querying", "url", log.URL, "startIndex", currentIndex, "endIndex", currentIndex+lookupSize, "endIndex", finalEntryIndex)
		entries := GetX509CertLogEntries(client, int64(currentIndex), int64(currentIndex+lookupSize))
		if !flags.includePrecert {
			ProcessDomainsFromEntries(entries)
		} else {
			ProcessDomainsFromEntriesIncludingPrecerts(entries)
		}

		currentIndex += uint64(len(entries))
		atomic.AddUint64(&totalProcessed, uint64(len(entries)))
		if atomic.LoadUint64(&totalProcessed) >= 10000 {
			SaveResume()
			atomic.StoreUint64(&totalProcessed, 0)
		}
		mep.Set(log.URL, int(currentIndex))
		//time.Sleep(550 * time.Millisecond)
	}
}
