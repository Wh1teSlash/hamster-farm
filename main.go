package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Tokens []string `yaml:"tokens"`
	Count  int      `yaml:"count"`
}

func readConfig(filename string) (*Config, error) {
	var config Config
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func doRequest(config *Config) {
	var wg sync.WaitGroup
	wg.Add(len(config.Tokens))
	for _, token := range config.Tokens {
		go func(token string) {
			defer wg.Done()
			body, _ := json.Marshal(map[string]interface{}{
				"count":         config.Count,
				"availableTaps": 0,
				"timestamp":     time.Now().UnixNano() / 1000000,
			})
			
			resBody := bytes.NewBuffer(body)
			
			req, err := http.NewRequest(http.MethodPost, "https://api.hamsterkombatgame.io/clicker/tap", resBody)
			
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", "Bearer "+token)
			
			if err != nil {
				log.Fatalf("Error creating request: %v", err)
			}
			
			client := &http.Client{}
			res, err := client.Do(req)
			if err != nil {
				log.Fatalf("Error performing request: %v", err)
			}
			defer res.Body.Close()
			
			responseBody, err := io.ReadAll(res.Body)
			if err != nil {
				log.Fatalf("Error reading response body: %v", err)
			}
			
			var responseMap map[string]interface{}
			err = json.Unmarshal(responseBody, &responseMap)
			if err != nil {
				log.Fatalf("Error unmarshaling response body: %v", err)
			}
			
			clickerUser, ok := responseMap["clickerUser"].(map[string]interface{})
			if !ok {
				log.Fatalf("Error: 'clickerUser' key does not exist or is not an object")
			}
			
			balance, ok := clickerUser["balanceCoins"].(float64)
			if !ok {
				log.Fatalf("Error: 'balanceCoins' key does not exist or is not a float64")
			}
			log.Printf("Balance: %f", balance)
		}(token)
	}
	wg.Wait()
}

func main() {
	config, err := readConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}
	
	doRequest(config)
	
	c := cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	c.AddFunc("@every 5m", func() { doRequest(config) })
	c.Start()
	select {}
}
