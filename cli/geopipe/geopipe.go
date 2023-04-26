package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/asaskevich/govalidator"
	"github.com/oschwald/maxminddb-golang"
	"github.com/rtfmkiesel/geopipe/pkg/dns"
	"github.com/rtfmkiesel/geopipe/pkg/maxmind"
	"github.com/rtfmkiesel/geopipe/pkg/utils"
)

// runnerOut() handels the maxmind.Result channel and prints the domain if the country code matches
func runnerOut(wg *sync.WaitGroup, chanJobs <-chan maxmind.Result, countrycode string) {
	defer wg.Done()

	// Slice to remember what was already printed
	var printed []string

	// For each job
	for job := range chanJobs {
		// If the countrycode matches & if it has no been printed yet
		if job.CountryCode == countrycode && !utils.Contains(printed, job.Domain) {
			// Print & append
			fmt.Println(job.Domain)
			printed = append(printed, job.Domain)
		}
	}
}

func main() {
	// Setup the args
	var flagCC string
	var flagDB string
	var flagResolvers string
	var flagThreads int
	var flagSilent bool
	// Parse the args
	flag.StringVar(&flagCC, "c", "US", "")
	flag.StringVar(&flagDB, "f", "./GeoLite2-Country.mmdb", "")
	flag.StringVar(&flagResolvers, "r", "9.9.9.9,1.1.1.1,8.8.8.8", "")
	flag.IntVar(&flagThreads, "t", 1, "")
	flag.BoolVar(&flagSilent, "s", false, "")
	flag.Usage = func() {
		fmt.Printf(`Usage: cat domains.txt | geopipe [OPTIONS]

Options:
    -c 	Two letter country code of the country to pipe thru (default: US)
    -f 	Path to the 'GeoLite2-Country.mmdb' file (default: ./GeoLite2-Country.mmdb)
    -r  Comma-separated list of DNS resolvers to use (default: 9.9.9.9,1.1.1.1,8.8.8.8)
    -t 	Number of threads to spawn (default: 1)
    -s  Do not print errors
    -h 	Prints this text
	
	`)
	}
	flag.Parse()

	// Check for the MMDB environment variable
	envDB, exists := os.LookupEnv("MMDB")
	if exists {
		// Override flag
		flagDB = envDB
	}

	// Set utils.Silent based on flag
	utils.Silent = flagSilent

	// Check if MaxMind database exists
	_, err := os.Stat(flagDB)
	if err != nil {
		utils.CatchCritErr(fmt.Errorf("file %s does not exist", flagDB))
	}

	// Open the MaxMind database
	mmDB, err := maxminddb.Open(flagDB)
	if err != nil {
		utils.CatchCritErr(err)
	}
	defer mmDB.Close()

	// Check that STDIN != empty
	stdinstat, err := os.Stdin.Stat()
	if err != nil {
		utils.CatchCritErr(err)
	}
	if stdinstat.Mode()&os.ModeNamedPipe == 0 {
		utils.CatchCritErr(fmt.Errorf("stdin was empty"))
	}

	// Create the channels
	chanDNSJobs := make(chan string)         // Jobs for the dns resolvers
	chanDBJobs := make(chan dns.Result)      // For the results of the DNS resolvers
	chanResults := make(chan maxmind.Result) // For the results of the MaxMind db lookup

	// Setup wait groups
	wgDNS := new(sync.WaitGroup)
	wgDB := new(sync.WaitGroup)
	wgOut := new(sync.WaitGroup)

	// Parse the DNS resolvers
	var dnsServers []string
	for _, resolver := range strings.Split(flagResolvers, ",") {
		// Validate if the given string is an IP
		if govalidator.IsIP(resolver) {
			// Append to the slice with ":53" added at the end
			dnsServers = append(dnsServers, fmt.Sprintf("%s:53", resolver))
		}
	}

	// Creating the DNS runners
	for i := 0; i < flagThreads; i++ {
		go dns.Runner(wgDNS, chanDNSJobs, chanDBJobs, dnsServers)
		wgDNS.Add(1)
	}

	// Creating the DB runners
	for i := 0; i < flagThreads; i++ {
		go maxmind.Runner(wgDB, chanDBJobs, chanResults, mmDB)
		wgDB.Add(1)
	}

	// Creating the output runner (only 1 needed)
	go runnerOut(wgOut, chanResults, flagCC)
	wgOut.Add(1)

	// Read from stdin
	stdin := bufio.NewScanner(os.Stdin)
	for stdin.Scan() {
		domain := stdin.Text()
		// Check if supplied domain is a valid DNSName to avoid later errors
		if !govalidator.IsDNSName(domain) {
			continue
		} else {
			// Add to the DNS jobs channel
			chanDNSJobs <- domain
		}
	}

	// If there was an error with stdin
	if err := stdin.Err(); err != nil {
		utils.CatchCritErr(err)
	}

	// Closing the DNS jobs channel is the start signal for the DNS runners
	close(chanDNSJobs)
	// Wait here for all the DNS channels to finish
	wgDNS.Wait()

	// Closing the DB jobs channel is the start signal for the DB runners
	close(chanDBJobs)
	// Wait here for all DB runners to finish
	wgDB.Wait()
	// Closing of the results channel is the start signal for the output runner
	close(chanResults)
	wgOut.Wait()
	os.Exit(0)
}
