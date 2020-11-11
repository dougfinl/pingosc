package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"gopkg.in/yaml.v2"
)

type host struct {
	Name  string `yaml:"name"`
	Addr  string `yaml:"ip"`
	OscUp string `yaml:"oscUp"`
	OscDn string `yaml:"oscDn"`
}

type appConfig struct {
	PingIntervalSeconds uint   `yaml:"pingInterval"`
	Hosts               []host `yaml:"hosts"`
}

var config appConfig

func main() {
	if len(os.Args) < 2 {
		fmt.Println("config file must be specified")
		os.Exit(1)
	}

	configFile, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("Failed to load config %v\n", err)
		os.Exit(1)
	}

	defer configFile.Close()

	decoder := yaml.NewDecoder(configFile)
	decoder.SetStrict(true)
	err = decoder.Decode(&config)
	if err != nil {
		fmt.Printf("Failed to parse config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(config)

	if len(config.Hosts) == 0 {
		fmt.Println("Please specify at least one host entry")
		os.Exit(1)
	}

	quit := make(chan struct{})

	// Catch ctrl+c
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		close(quit)
	}()

	ticker := time.NewTicker(time.Duration(config.PingIntervalSeconds) * time.Second)
	for {
		pingHosts()

		select {
		case <-ticker.C:
			continue
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func pingHosts() {
	fmt.Print(".")
}
