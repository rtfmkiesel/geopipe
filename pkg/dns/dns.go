package dns

import (
	"math/rand"
	"sync"
	"time"

	"github.com/projectdiscovery/retryabledns"
	"github.com/rtfmkiesel/geopipe/pkg/utils"
)

// Result for the DNS jobs
type Result struct {
	Domain string
	IP     string
}

// Go func to handle the DNS lookups since MaxMind needs IPs
//
// Will take input jobs in the form of strings and output results in the form of dns.Result
func Runner(wg *sync.WaitGroup, chanJobs <-chan string, chanOutput chan<- Result, dnsServers []string) {
	defer wg.Done()

	// Shuffle DNS resolvers
	r := rand.New(rand.NewSource(time.Now().Unix()))
	r.Shuffle(len(dnsServers), func(i, j int) {
		dnsServers[i], dnsServers[j] = dnsServers[j], dnsServers[i]
	})

	// Init DNS client
	dnsClient, err := retryabledns.New(dnsServers, 3)
	if err != nil {
		utils.CatchCritErr(err)
	}

	// For each job
	for domain := range chanJobs {
		// Make a DNS query
		addrs, err := dnsClient.Resolve(domain)
		if err != nil {
			utils.CatchErr(err)
			continue
		}

		// For each found IP addr
		for _, addr := range addrs.A {
			// Add a result to the output channel
			chanOutput <- Result{
				Domain: domain,
				IP:     addr,
			}
		}
	}
}
