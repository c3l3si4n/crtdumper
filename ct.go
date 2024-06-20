package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-resty/resty/v2"
	ct "github.com/google/certificate-transparency-go"
	ctClient "github.com/google/certificate-transparency-go/client"
	"github.com/google/certificate-transparency-go/jsonclient"
	ctLog "github.com/google/certificate-transparency-go/loglist3"
	"github.com/google/certificate-transparency-go/x509"
	"github.com/hashicorp/go-retryablehttp"
)

type CertData struct {
	LeafInput string `json:"leaf_input"`
	ExtraData string `json:"extra_data"`
}

type CertLog struct {
	Entries []CertData
}

func GetX509CertLogEntries(client *ctClient.LogClient, start, end int64) []ct.LogEntry {
	ctx := context.Background()
	resp, err := client.GetRawEntries(ctx, start, end)
	if err != nil {
		slog.Error("Failed get entries ", "log", client.BaseURI(), "startEntry", start, "endEntry", end, "err", err)
		return nil
	}
	entries := make([]ct.LogEntry, len(resp.Entries))
	for i, entry := range resp.Entries {
		index := start + int64(i)
		logEntry, err := ct.LogEntryFromLeaf(index, &entry)
		if x509.IsFatal(err) {
			slog.Error("Failed to parse entry", "log", client.BaseURI(), "entry", entry, "err", err)
			continue
		}
		entries[i] = *logEntry
	}
	return entries
}

func GetLogList() *ctLog.LogList {
	logListURL := ctLog.LogListURL
	client := resty.New()
	resp, err := client.R().
		EnableTrace().
		Get(logListURL)

	if err != nil {
		Abort(err.Error())
	}
	logList, err := ctLog.NewFromJSON(resp.Body())

	if err != nil {
		Abort(err.Error())
	}

	return logList

}

func CreateLogClient(log *ctLog.Log) *ctClient.LogClient {
	httpClient := retryablehttp.NewClient()
	httpClient.Logger = slog.Default()
	httpClient.RetryMax = 10

	httpClient.CheckRetry = func(c context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			slog.Error("", "err", err)
			bodyReader := resp.Body
			body := make([]byte, 1024)
			bodyReader.Read(body)
			slog.Error("", "body", string(body))
			return true, nil
		}
		if resp.StatusCode == 429 {
			// TODO: *Actually* increase sleep?
			slog.Warn("Got HTTP 429, increasing sleep for", "log", log.URL)
			return true, nil
		}

		if resp.StatusCode == 400 {
			slog.Warn("Got 400")
			bodyReader := resp.Body
			body := make([]byte, 1024)
			bodyReader.Read(body)
			slog.Warn("", "body", string(body))
		}
		// All other go to default policy
		return retryablehttp.DefaultRetryPolicy(c, resp, err)
	}
	stdHttpClient := httpClient.StandardClient()
	client, err := ctClient.New(log.URL, stdHttpClient, jsonclient.Options{})
	if err != nil {
		Abort(err.Error())
	}

	return client

}
