package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/go-ping/ping"
	"gopkg.in/yaml.v2"
)

const (
	pingResponseTimeout = 5 * time.Second
	pingPacketInterval  = 1 * time.Second
	packetLossThreshold = 50
)

type eosConsole struct {
	Addr string `yaml:"ip"`
	Port uint   `yaml:"port"`
}

type host struct {
	Name  string `yaml:"name"`
	Addr  string `yaml:"ip"`
	OscUp string `yaml:"oscUp"`
	OscDn string `yaml:"oscDn"`

	pinger *ping.Pinger
	isUp   bool
}

type appConfig struct {
	PingIntervalSeconds uint       `yaml:"pingInterval"`
	Console             eosConsole `yaml:"eosConsole"`
	Hosts               []host     `yaml:"hosts"`
}

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

	var config appConfig
	err = decoder.Decode(&config)
	if err != nil {
		fmt.Printf("Failed to parse config: %v\n", err)
		os.Exit(1)
	}

	if len(config.Hosts) == 0 {
		fmt.Println("Please specify at least one host entry")
		os.Exit(1)
	}

	fmt.Println("Running...")

	duration := time.Duration(config.PingIntervalSeconds) * time.Second
	ticker := time.NewTicker(duration)
	stop := make(chan struct{})

	go func() {
		for {
			createPingers(config.Hosts)
			runPingers(config.Hosts)
			printResults(config.Hosts)

			select {
			case <-ticker.C:
				continue
			case <-stop:
				ticker.Stop()
				return
			}
		}
	}()

	// Wait for SIGINT
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	fmt.Println("Stopping...")
}

func createPingers(hosts []host) []ping.Pinger {
	var pingers []ping.Pinger

	for i, host := range hosts {
		pinger, err := ping.NewPinger(host.Addr)
		if err != nil {
			fmt.Printf("Failed to create pinger for %v (%v)", host.Name, host.Addr)
			continue
		}

		hosts[i].pinger = pinger
		pinger.RecordRtts = false
		pinger.Interval = pingPacketInterval
		pinger.Count = 3
		pinger.Timeout = pingResponseTimeout
	}

	return pingers
}

func runPingers(hosts []host) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[string]bool)

	for _, host := range hosts {
		if host.pinger == nil {
			fmt.Println("pinger is null")
			continue
		}

		host.pinger.OnFinish = func(stats *ping.Statistics) {
			isUp := stats.PacketLoss <= packetLossThreshold
			mu.Lock()
			results[stats.Addr] = isUp
			mu.Unlock()
			wg.Done()
		}

		wg.Add(1)
		go host.pinger.Run()
	}

	wg.Wait()

	for i := range hosts {
		addr := hosts[i].Addr
		hosts[i].isUp = results[addr]
	}
}

func printResults(hosts []host) {
	for _, host := range hosts {
		var msg string
		if host.isUp {
			msg = "online"
		} else {
			msg = "offline"
		}
		fmt.Printf("%-20s%s\n", host.Name, msg)
	}

	fmt.Println("---------------------------")
}
