package maxmind

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/oschwald/maxminddb-golang"
	"github.com/rtfmkiesel/geopipe/pkg/dns"
)

type Result struct {
	Domain      string
	IP          string
	CountryCode string
}

// goroutine for MaxMindDB lookups
func Runner(wg *sync.WaitGroup, chanJobs <-chan dns.Result, chanOutput chan<- Result, db *maxminddb.Reader) {
	defer wg.Done()

	// for each job
	for job := range chanJobs {

		// struct for the country code inside the MaxMind db
		var mmdbRecord struct {
			Country struct {
				ISOCode string `maxminddb:"iso_code"`
			} `maxminddb:"country"`
		}

		// get the country code from the db
		err := db.Lookup(net.ParseIP(job.IP), &mmdbRecord)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// add a result to the output channel
		chanOutput <- Result{
			Domain:      job.Domain,
			IP:          job.IP,
			CountryCode: mmdbRecord.Country.ISOCode,
		}
	}
}
