package dns

import (
	"fmt"
	"os"
	"sync"

	"github.com/projectdiscovery/retryabledns"
)

type Result struct {
	Domain string
	IP     string
}

// goroutine for DNS lookups
func Runner(wg *sync.WaitGroup, chanJobs <-chan string, chanOutput chan<- Result, dnsServers []string) {
	defer wg.Done()

	// init DNS client
	dnsClient, err := retryabledns.New(dnsServers, 3)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// for each job
	for domain := range chanJobs {
		// make a DNS query
		addrs, err := dnsClient.Resolve(domain)
		if err != nil {
			// on timeout or non existing DNS entries
			continue
		}

		// for each found IP addr
		for _, addr := range addrs.A {
			// add a result to the output channel
			chanOutput <- Result{
				Domain: domain,
				IP:     addr,
			}
		}
	}
}
