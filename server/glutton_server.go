package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/coreos/go-iptables/iptables"

	"github.com/hectane/go-nonblockingchan"

	"github.com/synchroack/glutton"
	"github.com/synchroack/glutton/logger"
)

// SetIPTables modifies to iptables
func setIPTables() {
	ipt, err := iptables.New()
	if err != nil {
		panic(err)
	}
	ipt.Append("nat", "PREROUTING", "-p", "tcp", "--dport", "1:5000", "-j", "REDIRECT", "--to-port", "5000")
	ipt.Append("nat", "PREROUTING", "-p", "tcp", "--dport", "5002:65389", "-j", "REDIRECT", "--to-port", "5000")
	ipt.Append("nat", "PREROUTING", "-p", "udp", "-j", "REDIRECT", "--to-port", "5000")
}

func printLocalAddresses() {
	log.Println("Listening on the following interfaces:")
	interfaces, err := net.Interfaces()
	if err != nil {
		return
	}
	for _, interfaceObj := range interfaces {
		addresses, err := interfaceObj.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			switch value := address.(type) {
			case *net.IPNet:
				log.Printf("\t%v : %s (%s)\n", interfaceObj.Name, value, value.IP.DefaultMask())
			}
		}
	}
}

func main() {
	fmt.Println(`
	    _____ _       _   _
	   / ____| |     | | | |
	  | |  __| |_   _| |_| |_ ___  _ __
	  | | |_ | | | | | __| __/ _ \| '_ \
	  | |__| | | |_| | |_| || (_) | | | |
	   \_____|_|\__,_|\__|\__\___/|_| |_|

	`)

	logPath := flag.String("log", "/var/log/glutton/glutton.log", "Log path.")
	confPath := flag.String("conf", "/etc/glutton/ports.yml", "Config path.")
	setTables := flag.Bool("set-tables", false, "True to set iptables rules")
	capturePackets := flag.Bool("capture-packets", false, "True store pcap data")
	flag.Parse()

	// Print local addresses from interfaces
	printLocalAddresses()

	// Setup IPTables rules
	if *setTables {
		setIPTables()
	}

	// Setup file logging
	f, err := os.OpenFile(*logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(io.MultiWriter(f, os.Stdout))

	// Channel for TCP and UDP logging
	tcpChannel := nbc.New()
	udpChannel := nbc.New()

	// Load config file for remote services
	go glutton.LoadPorts(*confPath)

	var wg sync.WaitGroup

	if *capturePackets {
		log.Println("Starting Packet Capturing...")
		wg.Add(1)
		go logger.FindDevice(&wg)
	}

	// Load monitor modules (monitor.go)
	wg.Add(1)
	go glutton.MonitorTCPConnections(tcpChannel, &wg)
	log.Println("Initializing TCP connections tracking..")
	// Delay required for initialization of connection monitor (monitor.go) modules
	time.Sleep(3 * time.Second)

	wg.Add(1)
	go glutton.MonitorUDPConnections(udpChannel, &wg)
	log.Println("Initializing UDP connections tracking...")
	// Delay required for initialization of connection monitor (monitor.go) modules
	time.Sleep(3 * time.Second)

	// Load listeners modules (listeners.go)
	log.Println("Starting TCP Server...")
	wg.Add(1)
	go glutton.TCPListener(tcpChannel, &wg)

	log.Println("Starting UDP Server...")
	wg.Add(1)
	go glutton.UDPListener(udpChannel, &wg)

	wg.Wait()
}
