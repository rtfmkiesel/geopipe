<div align="center">
    <img src="https://gitlab.com/lu-ka/geopipe/-/raw/main/geopipe.png">
</div>

# geopipe

A tool to take domains from `stdin` and output them to `stdout` if have they at least one IP address inside the selected country.

## Requirements
You will require a `GeoLite2` database file. This database is free and can be downloaded [here](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data). License agreements of [MaxMind](https://maxmind.com) apply.


## Installation
```
go install gitlab.com/lu-ka/geopipe@latest
```

## Usage
```
usage: 'cat domains.txt | geopipe [OPTIONS]'

-c	Two letter country code of the country to pipe thru (default: US)
-f	Path to the 'GeoLite2-Country.mmdb' file (default: ./GeoLite2-Country.mmdb)
-o	Output mode {default, json, verbose} (default: default)
-w	Number of workers to spawn (default: 1)
-h	Prints this text
```

## Backstory
I wanted to do some _statistics_ on `.ch` domains. But since Swiss laws only apply to servers located in Switzerland, I needed a way to filter out domains with would have the correct TLD but are not located in Switzerland.

I've also never done something in GO before and since all the cool "pipe tools" are written in GO I thought I would teach myself something new.

## Kudos
```
# the <3 of this project
https://github.com/oschwald/maxminddb-golang 

# this was the base of this project and contributed some snippets here and there
https://github.com/thelicato/fire

# other noteworth modules include
https://github.com/projectdiscovery/retryabledns
https.//github.com/asaskevich/govalidator

# & all of the sub modules associated with the ones mentioned above
```

## License
This code is released under the [MIT License](https://gitlab.com/lu-ka/geopipe/blob/main/LICENSE).

## Legal
This code is provided for educational use only. If you engage in any illegal activity the author does not take any responsibility for it. By using this code, you agree with these terms.