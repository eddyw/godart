package main

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/goccy/go-yaml"
)

//go:embed jobs.yaml
var source []byte

type configuration struct {
	Jobs []struct {
		Name  string `yaml:"name"`
		Value string `yaml:"value"`
	} `yaml:"jobs"`
}

type result struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Digest string `json:"digest"`
}

func main() {
	workers := flag.Int("workers", 2, "Number of concurrent workers")
	flag.Parse()
	if *workers < 1 {
		panic("workers must be positive")
	}
	var config configuration
	if err := yaml.Unmarshal(source, &config); err != nil {
		panic(err)
	}
	directory, err := os.MkdirTemp("", "gart-config-worker-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(directory)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			http.Error(w, "gzip was not requested", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "application/json")
		writer := gzip.NewWriter(w)
		if err := json.NewEncoder(writer).Encode(map[string]string{"value": strings.ToUpper(r.URL.Query().Get("value"))}); err != nil {
			panic(err)
		}
		if err := writer.Close(); err != nil {
			panic(err)
		}
	}))
	defer server.Close()
	client := server.Client()
	defer client.CloseIdleConnections()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	jobs := make(chan int)
	output := make(chan result, len(config.Jobs))
	var group sync.WaitGroup
	for i := 0; i < *workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range jobs {
				job := config.Jobs[index]
				request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
				if err != nil {
					panic(err)
				}
				query := request.URL.Query()
				query.Set("value", job.Value)
				request.URL.RawQuery = query.Encode()
				response, err := client.Do(request)
				if err != nil {
					panic(err)
				}
				body, err := io.ReadAll(response.Body)
				if err != nil {
					panic(err)
				}
				if err := response.Body.Close(); err != nil {
					panic(err)
				}
				if response.StatusCode != http.StatusOK {
					panic(string(body))
				}
				file := filepath.Join(directory, fmt.Sprintf("%02d.json", index))
				if err := os.WriteFile(file, body, 0600); err != nil {
					panic(err)
				}
				stored, err := os.ReadFile(file)
				if err != nil {
					panic(err)
				}
				var decoded map[string]string
				if err := json.Unmarshal(stored, &decoded); err != nil {
					panic(err)
				}
				output <- result{Name: job.Name, Value: decoded["value"], Digest: fmt.Sprintf("%x", sha256.Sum256(stored))}
			}
		}()
	}
	for index := range config.Jobs {
		jobs <- index
	}
	close(jobs)
	group.Wait()
	close(output)
	var results []result
	for item := range output {
		results = append(results, item)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	encoded, err := json.Marshal(results)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(encoded))
}
