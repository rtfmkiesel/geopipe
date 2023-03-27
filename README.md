# geopipe
A tool to take domains from `stdin` and output them to `stdout` if have they at least one IP address inside the selected country.

## Requirements
You will need a `GeoLite2` database file. This database is free and can be downloaded [here](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data). License agreements of [MaxMind](https://maxmind.com) apply. Parse the path to this file via the `-f` option or with the environment variable `MMDB`.

## Usage
```
Usage: cat domains.txt | geopipe [OPTIONS]

Options:
    -c 	Two letter country code of the country to pipe thru (default: US)
    -f 	Path to the 'GeoLite2-Country.mmdb' file (default: ./GeoLite2-Country.mmdb)
    -t 	Number of threads to spawn (default: 1)
    -h 	Prints this text
```

## Installation
```bash
go install gitlab.com/rtfmkiesel/geopipe@latest
```

### Build from source
```bash
git clone https://gitlab.com/rtfmkiesel/geopipe
cd geopipe
# build binary
go build -ldflags="-s -w" "cli/geopipe"
# install into GOPATH/bin
go install "cli/geopipe"
```

## Kudos
### Contributors
- [cydave](https://github.com/cydave)

### Modules
- [oschwald/maxminddb-golang](https://github.com/oschwald/maxminddb-golang)