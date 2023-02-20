# geopipe

A tool to take domains from `stdin` and output them to `stdout` if have they at least one IP address inside the selected country.

## Requirements
You will require a `GeoLite2` database file. This database is free and can be downloaded [here](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data). License agreements of [MaxMind](https://maxmind.com) apply.

Parse the path to this file via the `-f` option or with the environment variable `MMDB`.

## Installation
```bash
go install gitlab.com/rtfmkiesel/geopipe@latest
```

## Build from source
```bash
git clone https://gitlab.com/rtfmkiesel/geopipe
cd geopipe
go build
```

## Usage
```
usage: 'cat domains.txt | geopipe [OPTIONS]'

-c	Two letter country code of the country to pipe thru (default: US)
-f	Path to the 'GeoLite2-Country.mmdb' file (default: ./GeoLite2-Country.mmdb)
-o	Output mode {default, json, verbose} (default: default)
-w	Number of workers to spawn (default: 1)
-r	Comma separated list of DNS resolvers (w/ ports) to use (default: 1.1.1.1:53,8.8.8.8:53,9.9.9.9:53)
-h	Prints this text
```

## Kudos
### Contributors
- [cydave](https://gitlab.com/cydave)

### Modules & Code Snippets
```
# the <3 of this project
https://github.com/oschwald/maxminddb-golang 

# this was the base of this project and contributed some snippets here and there
https://github.com/thelicato/fire

# other noteworthy modules include
https://github.com/projectdiscovery/retryabledns
https.//github.com/asaskevich/govalidator

# & all of the sub modules associated with the ones mentioned above
```

## Why
I wanted to do some _statistics_ on `.ch` domains. But since Swiss laws only apply to servers located in Switzerland, I needed a way to filter out domains which would have the correct TLD but are not located in Switzerland.

## License
This code is released under the [MIT License](https://gitlab.com/rtfmkiesel/geopipe/blob/main/LICENSE).

## Legal
This code is provided for educational use only. If you engage in any illegal activity the author does not take any responsibility for it. By using this code, you agree with these terms.