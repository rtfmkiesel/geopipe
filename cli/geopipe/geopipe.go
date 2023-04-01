package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/asaskevich/govalidator"
	"github.com/oschwald/maxminddb-golang"
	"gitlab.com/rtfmkiesel/geopipe/pkg/dns"
	"gitlab.com/rtfmkiesel/geopipe/pkg/maxmind"
)

// returns true if []string slice contains string
func contains(list []string, query string) bool {
	for _, item := range list {
		if item == query {
			return true
		}
	}

	return false
}

// goroutine for printing the results
func runnerOut(wg *sync.WaitGroup, chanJobs <-chan maxmind.Result, countrycode string) {
	defer wg.Done()

	// slice to remember what was already printed
	var printed []string

	// for each job
	for job := range chanJobs {
		// if the countrycode matches
		if job.CountryCode == countrycode {
			// if it has no been printed yet
			if !contains(printed, job.Domain) {
				// print & append
				fmt.Println(job.Domain)
				printed = append(printed, job.Domain)
			}
		}
	}
}

func main() {
	// setup the args
	var flagCC string
	var flagDB string
	var flagThreads int
	// parse the args
	flag.StringVar(&flagCC, "c", "US", "")
	flag.StringVar(&flagDB, "f", "./GeoLite2-Country.mmdb", "")
	flag.IntVar(&flagThreads, "t", 1, "")
	flag.Usage = func() {
		fmt.Printf(`Usage: cat domains.txt | geopipe [OPTIONS]

Options:
    -c 	Two letter country code of the country to pipe thru (default: US)
    -f 	Path to the 'GeoLite2-Country.mmdb' file (default: ./GeoLite2-Country.mmdb)
    -t 	Number of threads to spawn (default: 1)
    -h 	Prints this text
	
	`)
	}
	flag.Parse()

	// check for the MMDB environment variable
	envDB, exists := os.LookupEnv("MMDB")
	if exists {
		flagDB = envDB
	}

	// check if dbpath exists
	_, err := os.Stat(flagDB)
	if err != nil {
		fmt.Printf("file %s does not exist\n", flagDB)
		os.Exit(1)
	}

	// open the maxmind database
	mmDB, err := maxminddb.Open(flagDB)
	if err != nil {
		log.Fatal(err)
	}
	defer mmDB.Close()

	// check that STDIN != empty
	stdinstat, err := os.Stdin.Stat()
	if err != nil {
		log.Fatal(err)
	}
	if stdinstat.Mode()&os.ModeNamedPipe == 0 {
		fmt.Printf("stdin was empty\n")
		os.Exit(1)
	}

	// create the channels
	chanDNSJobs := make(chan string)         // jobs for the dns resolvers
	chanDBJobs := make(chan dns.Result)      // jobs to look up inside the MaxMind db
	chanResults := make(chan maxmind.Result) // for the results of the MaxMind db lookup

	// setup wait groups
	wgDNS := new(sync.WaitGroup)
	wgDB := new(sync.WaitGroup)
	wgOut := new(sync.WaitGroup)

	// populating resolver pool
	dnsServers := []string{"9.9.9.9:53", "1.1.1.1:53", "8.8.8.8:53"}

	// creating the DNS runners
	for i := 0; i < flagThreads; i++ {
		go dns.Runner(wgDNS, chanDNSJobs, chanDBJobs, dnsServers)
		wgDNS.Add(1)
	}

	// creating the DB runners
	for i := 0; i < flagThreads; i++ {
		go maxmind.Runner(wgDB, chanDBJobs, chanResults, mmDB)
		wgDB.Add(1)
	}

	// creating the output runner
	go runnerOut(wgOut, chanResults, flagCC)
	wgOut.Add(1)

	// read from stdin
	stdin := bufio.NewScanner(os.Stdin)
	for stdin.Scan() {
		domain := stdin.Text()
		// check if supplied domain is a valid DNSName to avoid later errors
		if !govalidator.IsDNSName(domain) {
			continue
		} else {
			// add to the DNS jobs channel
			chanDNSJobs <- domain
		}
	}

	// if there was an error with stdin
	if err := stdin.Err(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// closing the DNS jobs channel is the start signal for the DNS runners
	close(chanDNSJobs)
	// wait here for all the DNS channels to finish
	wgDNS.Wait()

	// closing the DB jobs channel is the start signal for the DB runners
	close(chanDBJobs)
	// wait here for all DB runners to finish
	wgDB.Wait()
	// closing of the results channel is the start signal for the output runner
	close(chanResults)
	wgOut.Wait()
	os.Exit(0)
}
