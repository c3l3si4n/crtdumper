package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	stdLog "log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/alphadose/haxmap"
	"github.com/go-resty/resty/v2"
	ct "github.com/google/certificate-transparency-go"
	ctClient "github.com/google/certificate-transparency-go/client"
	"github.com/google/certificate-transparency-go/jsonclient"
	ctLog "github.com/google/certificate-transparency-go/loglist3"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

var mep = haxmap.New[string, int]()

var LookUpSizeFlag uint64 = 20
type CachedLogProgress struct {
	LogURL string `json:"log_url"`
	Index  int    `json:"index"`
}

type MapCache struct {
	Entries []CachedLogProgress `json:"entries"`
}

func SaveCache() {
	serializedMap := MapCache{}
	mep.ForEach(func(key string, value int) bool {
		entry := CachedLogProgress{LogURL: key, Index: value}

		serializedMap.Entries = append(serializedMap.Entries, entry)
		return true // return `true` to continue iteration and `false` to break iteration
	})

	file, err := os.OpenFile("cache.json", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		Abort(err.Error())
	}
	defer file.Close()
	serialized, err := json.Marshal(serializedMap)
	if err != nil {
		Abort(err.Error())
	}
	file.Write(serialized)

	log.Printf("Saved cache to cache.json")

}

func LoadCache() {
	file, err := os.OpenFile("cache.json", os.O_RDONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	serializedMap := MapCache{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&serializedMap)
	if err != nil {
		Abort(err.Error())
	}
	for _, entry := range serializedMap.Entries {
		mep.Set(entry.LogURL, entry.Index)
	}
	mep.ForEach(func(key string, value int) bool {
		log.Printf("Loaded %s at position %d", key, value)
		return true // return `true` to continue iteration and `false` to break iteration
	})
}

func Abort(msg string) {
	log.Println(msg)
	SaveCache()
	panic(msg)
}

var outputFilename = ""

func SaveToOutput(data string) {
	file, err := os.OpenFile(outputFilename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		Abort(err.Error())
	}
	defer file.Close()
	file.Write([]byte(data))

}

func main() {
	shouldLoadCache := flag.Bool("r", false, "Resume from last execution")
	flag.Uint64Var(&LookUpSizeFlag, "n", 20, "number of records to query at once")
	flag.StringVar(&outputFilename, "o", "", "Output filename")

	flag.Parse()

	if outputFilename == "" {
		Abort("Output filename is required. Specify it with -o flag.")
	}

	if *shouldLoadCache {
		LoadCache()
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		Abort("Interrupted by user.")

	}()

	logList := GetLogList()
	var wg sync.WaitGroup

	for _, log := range logList.Operators {
		wg.Add(1)

		for _, log := range log.Logs {
			wg.Add(1)

			go func(log *ctLog.Log) {
				defer wg.Done()

				if log.State.LogStatus() != ctLog.UsableLogStatus {
					fmt.Printf("Skipping log %s due to status %s", log.URL, log.State.LogStatus())
					return
				}

				client := CreateLogClient(log)

				STH, err := client.GetSTH(context.Background())

				if err != nil {
					Abort(err.Error())
				}

				finalEntryIndex := STH.TreeSize

				currentIndexInt, _ := mep.Get(log.URL)
				lookupSize := LookUpSizeFlag
				currentIndex := uint64(currentIndexInt)
				for currentIndex < finalEntryIndex {
					if currentIndex+lookupSize > finalEntryIndex {
						lookupSize = finalEntryIndex - currentIndex
					}

					fmt.Println("Querying", log.URL, "from", currentIndex, "to", currentIndex+lookupSize, "with size", finalEntryIndex)
					entries := GetLogEntries(client, int64(currentIndex), int64(currentIndex+lookupSize))

					for _, entry := range entries {
						if entry.X509Cert != nil {
							commonName := GetDomainsFromEntry(entry)
							if commonName != "" {
								SaveToOutput(commonName + "\n")
							}
						}
					}

					currentIndex += uint64(len(entries))
					mep.Set(log.URL, int(currentIndex))

					//time.Sleep(550 * time.Millisecond)
				}
			}(log)

		}

	}
	wg.Wait()
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
	//httpClient.Logger = stdLog.New(stdLog.Writer(), "http: ", stdLog.LstdFlags)
	// httpClient.Logger = nil
	httpClient.RetryMax = 10

	httpClient.CheckRetry = func(c context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			stdLog.Println("Error:", err)
			bodyReader := resp.Body
			body := make([]byte, 1024)
			bodyReader.Read(body)
			stdLog.Println("Body:", string(body))
			return true, nil
		}
		if resp.StatusCode == 429 {
			stdLog.Println("Got 429, increasing sleep amount!")
			return true, nil
		}

		if resp.StatusCode == 400 {
			stdLog.Println("Got 400")
			bodyReader := resp.Body
			body := make([]byte, 1024)
			bodyReader.Read(body)
			stdLog.Println("Body:", string(body))
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

func GetLogEntries(client *ctClient.LogClient, start, end int64) []ct.LogEntry {
	context := context.Background()

	entries, err := client.GetEntries(context, start, end)
	if err != nil {
		stdLog.Printf("Getting entries from %d to %d on %s failed.\n", start, end, client.BaseURI())

		Abort(err.Error())
	}

	return entries
}

type CertData struct {
	LeafInput string `json:"leaf_input"`
	ExtraData string `json:"extra_data"`
}

type CertLog struct {
	Entries []CertData
}

func GetDomainsFromEntry(entry ct.LogEntry) string {
	if entry.X509Cert.Subject.CommonName != "flowers-to-the-world.com" {
		return entry.X509Cert.Subject.CommonName
	}
	return ""

}

