package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/alphadose/haxmap"
)

var mep = haxmap.New[string, int]()

type CachedLogProgress struct {
	LogURL string `json:"log_url"`
	Index  int    `json:"index"`
}

type MapCache struct {
	Entries []CachedLogProgress `json:"entries"`
}

func SaveResume() {
	serializedMap := MapCache{}
	mep.ForEach(func(key string, value int) bool {
		entry := CachedLogProgress{LogURL: key, Index: value}

		serializedMap.Entries = append(serializedMap.Entries, entry)
		return true // return `true` to continue iteration and `false` to break iteration
	})

	file, err := os.OpenFile(flags.resumeFilename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		Abort(err.Error())
	}
	defer file.Close()
	serialized, err := json.Marshal(serializedMap)
	if err != nil {
		Abort(err.Error())
	}
	file.Write(serialized)

	slog.Info("Saved cache", "filename", flags.resumeFilename)
}

func LoadResume() {
	file, err := os.OpenFile(flags.resumeFilename, os.O_RDONLY, 0644)
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
		slog.Info(fmt.Sprintf("Loaded %s at position %d", key, value))
		return true // return `true` to continue iteration and `false` to break iteration
	})
}
