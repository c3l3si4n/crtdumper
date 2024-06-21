package main

import (
	"log/slog"
	"sync"
)

var domainChannel = make(chan string)

func main() {

	slog.Debug("Getting operator and log metadata")
	logList := GetLogList()

	if flags.actionList {
		slog.Debug("Action list")
		doList(logList)
	}

	if flags.resume {
		slog.Debug("Loading resume data")
		LoadResume()
	}

	var wg sync.WaitGroup
	slog.Debug("Starting output writer coroutine")
	wg.Add(1)
	go DomainChannelToFile(&wg)
	slog.Debug("Starting log scrapers")
	for _, operator := range logList.Operators {
		if SkipOperator(operator) {
			continue
		}
		for _, log := range operator.Logs {
			if SkipLog(log) {
				continue
			}
			wg.Add(1)
			go ProcessLog(&wg, log, operator)
		}
	}
	slog.Info("Waiting for WaitGroup")
	wg.Wait()
	slog.Info("Closing domainChannel")
	close(domainChannel)
}
