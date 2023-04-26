package maxmind

import (
	"net"
	"sync"

	"github.com/oschwald/maxminddb-golang"
	"github.com/rtfmkiesel/geopipe/pkg/dns"
	"github.com/rtfmkiesel/geopipe/pkg/utils"
)

// Struct for the DB lookup results
type Result struct {
	Domain      string
	IP          string
	CountryCode string
}

// Struct for the country code inside the MaxMind db
type mmdbRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

// Go func to handle the MaxMind DB lookups
//
// Will take input jobs in the form of dns.Result and output results in the form of maxmind.Result
func Runner(wg *sync.WaitGroup, chanJobs <-chan dns.Result, chanOutput chan<- Result, db *maxminddb.Reader) {
	defer wg.Done()

	// For each job
	for job := range chanJobs {

		// Get the country code from the db
		var record mmdbRecord
		err := db.Lookup(net.ParseIP(job.IP), &record)
		if err != nil {
			utils.CatchErr(err)
			continue
		}

		// Add a result to the output channel
		chanOutput <- Result{
			Domain:      job.Domain,
			IP:          job.IP,
			CountryCode: record.Country.ISOCode,
		}
	}
}
