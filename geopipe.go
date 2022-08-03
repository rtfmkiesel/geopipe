package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/oschwald/maxminddb-golang"
	"github.com/projectdiscovery/retryabledns"
	"golang.org/x/exp/slices"
)

// prints some information if mode == verbose
func verbose(mode string, msg string) {

	if mode == "verbose" {
		fmt.Println(msg)
	}
}

// banner
func banner(mode string) {

	if mode == "verbose" {
		fmt.Println(`
  ▄████ ▓█████  ▒█████   ██▓███   ██▓ ██▓███  ▓█████ 
 ██▒ ▀█▒▓█   ▀ ▒██▒  ██▒▓██░  ██▒▓██▒▓██░  ██▒▓█   ▀ 
▒██░▄▄▄░▒███   ▒██░  ██▒▓██░ ██▓▒▒██▒▓██░ ██▓▒▒███   
░▓█  ██▓▒▓█  ▄ ▒██   ██░▒██▄█▓▒ ▒░██░▒██▄█▓▒ ▒▒▓█  ▄ 
░▒▓███▀▒░▒████▒░ ████▓▒░▒██▒ ░  ░░██░▒██▒ ░  ░░▒████▒
 ░▒   ▒ ░░ ▒░ ░░ ▒░▒░▒░ ▒▓▒░ ░  ░░▓  ▒▓▒░ ░  ░░░ ▒░ ░`)
		fmt.Print("by \033[36m@lukahacksstuff\033[0m | \033[36mhttps://gitlab.com/lu-ka/geopipe\033[0m\n\n")
	}
}

// usage/help
func usage(error string) {

	banner("verbose")
	if error != "" {
		fmt.Printf("\033[31m%s\033[0m\n\n", error)
	}
	fmt.Println("usage: 'cat domains.txt | geopipe [OPTIONS]'")
	fmt.Print(`
-c	Two letter country code of the country to pipe thru (default: US)
-f	Path to the 'GeoLite2-Country.mmdb' file (default: ./GeoLite2-Country.mmdb)
-o	Output mode {default, json, verbose} (default: default)
-w	Number of workers to spawn (default: 1)
-h	Prints this text
`)
}

// DNS resolver
func lookupDNS(domain string, dnsClient retryabledns.Client, workerid int, mode string) []string {

	verbose(mode, fmt.Sprintf("[DNS worker %d] Resolving %s", workerid, domain))

	// get the IP from the DNS
	dnsData, err := dnsClient.Resolve(domain)
	if err != nil {
		// on timeout or non existing DNS entries
		verbose(mode, fmt.Sprintf("[DNS worker %d] ERROR while resolving %s", workerid, domain))
		return nil
	}

	return dnsData.A
}

// worker for concurrent DNS lookups
func workerDNS(wgDNS *sync.WaitGroup, chanDNSJobs <-chan string, chanDBJobs chan<- string, workerid int, mode string) {

	defer wgDNS.Done()

	// init DNS client
	// since the MaxMind db wants the ip and not the domain name
	resolvers := []string{"9.9.9.9:53", "1.1.1.1:53", "8.8.8.8:53"}
	dnsClient, err := retryabledns.New(resolvers, 3)
	if err != nil {
		log.Fatal(err)
	}

	// iterate thru all DNS jobs
	for domain := range chanDNSJobs {
		resultDNS := lookupDNS(domain, *dnsClient, workerid, mode)
		for _, ip := range resultDNS {
			// return semicolon separated string
			job := fmt.Sprintf("%s;%s", domain, ip)
			chanDBJobs <- job
		}
	}

	verbose(mode, fmt.Sprintf("[DNS worker %d] Done", workerid))
}

// MaxMind DB lookup
func lookupDB(strIP string, countrycode string, workerid int, db *maxminddb.Reader, mode string) string {

	verbose(mode, fmt.Sprintf("[DB worker %d] Comparing %s", workerid, strIP))

	// convert the ip from string to net.IP
	ip := net.ParseIP(strIP)

	// get the country code from the db
	err := db.Lookup(ip, &record)
	if err != nil {
		log.Panic(err)
	}

	// if the country code matches
	if record.Country.ISOCode == countrycode {

		verbose(mode, fmt.Sprintf("\033[32m[DB worker %d] %s matches\033[0m", workerid, strIP))
		// return the found country code and true as semicolon separated string
		return fmt.Sprintf("%s;%t", record.Country.ISOCode, true)

	} else { // if the country code does not match

		verbose(mode, fmt.Sprintf("[DB worker %d] %s does not match", workerid, strIP))
		// return the found country code and false as semicolon separated string
		return fmt.Sprintf("%s;%t", record.Country.ISOCode, false)

	}
}

// worker for concurrent DB lookups
func workerDB(wgDB *sync.WaitGroup, chanDBJobs <-chan string, chanResults chan<- string, workerid int, countrycode string, db *maxminddb.Reader, mode string) {

	defer wgDB.Done()

	// iterate thru all jobs
	for job := range chanDBJobs {
		// split up the job by ";"
		domain := strings.Split(job, ";")[0]
		ip := strings.Split(job, ";")[1]

		// lookup up the ip inside the MaxMind db
		resultDB := lookupDB(ip, countrycode, workerid, db, mode)

		// split up the result from the db lookup by ";"
		countrycountDB := strings.Split(resultDB, ";")[0]
		matchDB := strings.Split(resultDB, ";")[1]

		// send the final job result as a semicolon separated string to the output channel
		result := fmt.Sprintf("%s;%s;%s;%s", domain, ip, countrycountDB, matchDB)
		chanResults <- result
	}

	// no more jobs, worker is done
	verbose(mode, fmt.Sprintf("[DB worker %d] Done", workerid))
}

// worker for printing the final results
func workerOUT(chanResults <-chan string, chanDBHits chan<- int, countrycode string, mode string) {

	// struct for the results
	type strResult struct {
		Domain      string `json:"domain"`
		IP          string `json:"ip_address"`
		Countrycode string `json:"country_code"`
		Match       bool   `json:"match"`
	}

	// slice of matched domains
	var slMatched []string
	hitcounter := 0

	// read all the results as they come in
	for rawresult := range chanResults {

		// convert the last part to a bool
		match, _ := strconv.ParseBool(strings.Split(rawresult, ";")[3])

		// append a result to the slice of results
		result := strResult{
			Domain:      strings.Split(rawresult, ";")[0],
			IP:          strings.Split(rawresult, ";")[1],
			Countrycode: strings.Split(rawresult, ";")[2],
			Match:       match,
		}

		// if the user wants json output
		if mode == "json" {
			jsonresult, err := json.Marshal(result)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(string(jsonresult))

		} else { // normal mode, just output if the domain matched

			// check if the domain matched and if is has not been printed already via slMatched
			if result.Match && !slices.Contains(slMatched, result.Domain) {
				// only print in default mode
				if mode == "default" {
					fmt.Println(result.Domain)
				}
				hitcounter++
				slMatched = append(slMatched, result.Domain)
			}
		}

	}

	verbose(mode, "[OUT worker 0] Done")

	// send the amount of hits back to main when were done
	chanDBHits <- hitcounter
}

// struct for the country code inside the MaxMind db
var record struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

func main() {

	// to calculate the runtime at the end
	timeStart := time.Now()

	// args (flags)
	var countrycode string
	var dbpath string
	var mode string
	var workercount int
	var help bool

	flag.StringVar(&countrycode, "c", "US", "Two letter country code of the country to pipe thru")
	flag.StringVar(&dbpath, "f", "./GeoLite2-Country.mmdb", "Path to the 'GeoLite2-Country.mmdb' file")
	flag.StringVar(&mode, "o", "default", "Output mode {default, json, verbose}")
	flag.IntVar(&workercount, "w", 1, "Number of workers to spawn")
	flag.BoolVar(&help, "h", false, "Prints this text")
	flag.Parse()

	// show help
	if help {
		usage("")
		os.Exit(0)
	}

	// check for unknown mode
	switch mode {
	case "default", "json", "verbose":
		// known mode
	default:
		// unknown mode
		usage(fmt.Sprintf("[main] Unknown mode %s", mode))
		os.Exit(1)
	}

	// check if env var exists
	dbpathENV, ok := os.LookupEnv("MMDB")
	if ok {
		dbpath = dbpathENV
		verbose(mode, "[main] Got dbpath from ENV")
	}

	// check if dbpath exists
	_, err := os.Stat(dbpath)
	if err != nil {
		usage(fmt.Sprintf("[main] File %s does not exist", dbpath))
		os.Exit(1)
	}
	verbose(mode, fmt.Sprintf("[main] File %s exists", dbpath))

	banner(mode)
	verbose(mode, "[main] Mode: Verbose")

	// open the maxminddb
	verbose(mode, "[main] Opening MaxMind DB")
	db, err := maxminddb.Open(dbpath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	verbose(mode, "[main] Successfully opened MaxMind DB")

	// check that STDIN != empty
	stdinstat, err := os.Stdin.Stat()
	if err != nil {
		log.Fatal(err)
	}

	if stdinstat.Mode()&os.ModeNamedPipe == 0 {
		usage("[main] STDIN was empty")
		os.Exit(1)
	}

	// create the channels
	chanDNSJobs := make(chan string) // jobs for the dns resolvers
	chanDBJobs := make(chan string)  // jobs to look up inside the MaxMind db
	chanResults := make(chan string) // for the results of the MaxMind db lookup
	chanDBHits := make(chan int, 1)  // for the final statistics

	// wait groups
	wgDNS := new(sync.WaitGroup)
	wgDB := new(sync.WaitGroup)

	// creating the DNS workers
	for workerid := 0; workerid < workercount; workerid++ {
		go workerDNS(wgDNS, chanDNSJobs, chanDBJobs, workerid, mode)
		wgDNS.Add(1)
	}
	verbose(mode, fmt.Sprintf("[main] Created %d DNS worker(s)", workercount))

	// creating the DB workers
	for workerid := 0; workerid < workercount; workerid++ {
		go workerDB(wgDB, chanDBJobs, chanResults, workerid, countrycode, db, mode)
		wgDB.Add(1)
	}
	verbose(mode, fmt.Sprintf("[main] Created %d DB worker(s)", workercount))

	// creating the output worker
	go workerOUT(chanResults, chanDBHits, countrycode, mode)
	verbose(mode, "[main] Created 1 Output worker(s)")

	// get the domains from stdin
	verbose(mode, "[main] Reading data from STDIN")

	inputdomaincounter := 0
	stdin := bufio.NewScanner(os.Stdin)
	for stdin.Scan() {
		// check if supplied domain is a valid DNSName to avoid later errors
		if !govalidator.IsDNSName(stdin.Text()) {
			verbose(mode, fmt.Sprintf("\033[33m[main] %s is not valid domain, ignoring\033[0m", stdin.Text()))
			continue
		} else {
			if len(stdin.Text()) == 0 {
				break
			}
			chanDNSJobs <- stdin.Text()
			inputdomaincounter++
		}
	}

	// if there was an error with STDIN
	if err := stdin.Err(); err != nil {
		usage("[main] Error while reading STDIN")
		os.Exit(1)
	}

	// closing the DNS Jobs channel means that workerDB know when there are no more jobs to do
	verbose(mode, fmt.Sprintf("[main] Got a total of %d valid domains from STDIN", inputdomaincounter))
	close(chanDNSJobs)
	// wait here for all the DNS channels to finish
	wgDNS.Wait()
	close(chanDBJobs)

	// wait here for all DB workers to finish
	wgDB.Wait()
	close(chanResults)

	// read the hitcounter from the workerOUT
	hitcounter := <-chanDBHits
	close(chanDBHits)

	if hitcounter == 0 {
		// protection for divide by zero
		verbose(mode, fmt.Sprintf("[main] Zero of the domains supplied have at least one DNS entry pointing to an IP in %s", countrycode))
	} else {
		percent := ((hitcounter * 100) / inputdomaincounter)
		verbose(mode, fmt.Sprintf("[main] %d%% of the domains supplied have at least one DNS entry pointing to an IP in %s", percent, countrycode))
	}

	// runtime
	timeElapsed := time.Since(timeStart)

	verbose(mode, fmt.Sprintf("[main] Completed in %s", timeElapsed))
	os.Exit(0)
}
