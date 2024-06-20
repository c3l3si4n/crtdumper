package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	ctLog "github.com/google/certificate-transparency-go/loglist3"
)

func Abort(msg string) {
	slog.Warn(msg)
	SaveCache()
	panic(msg)
}

func DomainChannelToFile(wg *sync.WaitGroup) {
	defer wg.Done()
	file, err := os.OpenFile(flags.outputFilename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		Abort(err.Error())
	}

	defer file.Close()

	for domain := range domainChannel {
		file.WriteString(domain + "\n")
	}
}

func DomainsToOutputStdout(values []string) {
	for _, value := range values {
		fmt.Println(value)
	}
}
func DomainsToOutputFile(values []string) {
	for _, value := range values {
		domainChannel <- fmt.Sprint(value)
	}
}

func DomainToOutputFile(value string) {
	domainChannel <- fmt.Sprint(value)
}

func DomainToOutputStdout(value string) {
	fmt.Println(value)
}

func SkipOperator(operator *ctLog.Operator) bool {
	if len(flags.onlyOperator) > 0 && strings.EqualFold(operator.Name, flags.onlyOperator) {
		return true
	}
	if len(flags.operatorsToSkip) > 0 {
		for _, operatorToSkip := range flags.operatorsToSkip {
			if strings.EqualFold(operator.Name, operatorToSkip) {
				slog.Info("Skipping operator on request", "operator", operator.Name)
				return true
			}
		}
	}
	return false
}
func SkipLog(log *ctLog.Log) bool {
	if len(flags.onlyLog) > 0 && !strings.EqualFold(log.URL, flags.onlyLog) {
		return true
	}
	if len(flags.logsToSkip) > 0 {
		for _, logToSkip := range flags.logsToSkip {
			if strings.EqualFold(log.URL, logToSkip) {
				slog.Info("Skipping log on request", "log", log.URL)
				return true
			}
			if log.State.LogStatus() != ctLog.UsableLogStatus {
				slog.Info("Skipping because of state", "log", log.URL, "status", log.State.LogStatus())
				return true
			}
		}
	}
	return false
}
