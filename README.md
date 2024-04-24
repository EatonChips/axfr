# axfr

AXFR does what it says, conducts zone transfers on domains against their authoritative name servers.

## Installation

```
go install github.com/eatonchips/axfr@latest
```

```
git clone https://github.com/eatonchips/axfr
cd axfr
go install
```

## Usage

```
Usage of axfr:
  -c, --csv string            Output file for csv format
  -d, --domains stringArray   Domain names to transfer
  -f, --file string           File containing domain names to transfer
  -j, --json string           Output file for json format
  -n, --nameserver string     DNS Server for resolving domain name servers
  -o, --output string         Output format (json,csv)
  -v, --verbose               Verbose output
```

## Attempt transfer on domains from command line

```
axfr -d zonetransfer.me
```

```
$ axfr -d zonetransfer.me -d example.com
[*] Using nameserver: 10.0.0.1
[*] Attempting zone transfer for 2 domains
[+] Zone transfer successful for zonetransfer.me against nsztm1.digi.ninja, identified 50 records
[+] Zone transfer successful for zonetransfer.me against nsztm2.digi.ninja, identified 51 records
[-] Zone transfer failed for example.com against b.iana-servers.net: dns: bad xfr rcode: 5
[-] Zone transfer failed for example.com against a.iana-servers.net: dns: bad xfr rcode: 9
```

## From file

```
axfr -f domains.txt
```

Domains provided via file and cli are combined.
```
axfr -f domains.txt -d zonetransfer.me
```

## Output Formats

```
$ axfr -d zonetransfer.me -o csv,json
[*] Using nameserver: 1.1.1.1
[*] Using json output file: axfr-20240424-155029.json
[*] Using csv output file: axfr-20240424-155029.csv
[*] Attempting zone transfer for 1 domains
[+] Zone transfer successful for zonetransfer.me against nsztm2.digi.ninja, identified 51 records
[+] Zone transfer successful for zonetransfer.me against nsztm1.digi.ninja, identified 50 records
```

```
$ axfr -d zonetransfer.me -o csv,json --csv output.csv --json output.json
[*] Using nameserver: 1.1.1.1
[*] Using json output file: output.json
[*] Using csv output file: output.csv
[*] Attempting zone transfer for 1 domains
[+] Zone transfer successful for zonetransfer.me against nsztm1.digi.ninja, identified 50 records
[+] Zone transfer successful for zonetransfer.me against nsztm2.digi.ninja, identified 51 records
```

## Specify NS lookup nameserver

```
$ axfr -d zonetransfer.me -n 8.8.8.8
[*] Using nameserver: 8.8.8.8
[*] Attempting zone transfer for 1 domains
[+] Zone transfer successful for zonetransfer.me against nsztm2.digi.ninja, identified 51 records
[+] Zone transfer successful for zonetransfer.me against nsztm1.digi.ninja, identified 50 records
```

## Use specific domain nameservers

It can be useful to test zone transfers against other nameservers that may be indicated by current DNS NS records, such as historical nameservers that have not been decommisioned (https://dnshistory.org/). These can be specified by adding an `@` symbol to the domain, either in the CLI flag or the file.

```
$ cat domains-file 
zonetransfer.me
zonetransfer.me@hal.ns.cloudflare.com
zonetransfer.me@lisa.ns.cloudflare.com
example.com@1.1.1.1

$ axfr -f domains-file 
[*] Reading domains from file
[*] Using nameserver: 10.0.0.1
[*] Attempting zone transfer for 4 domains
[+] Zone transfer successful for zonetransfer.me against nsztm1.digi.ninja, identified 50 records
[+] Zone transfer successful for zonetransfer.me against nsztm2.digi.ninja, identified 51 records
[-] Zone transfer failed for zonetransfer.me against hal.ns.cloudflare.com: dns: bad xfr rcode: 1
[-] Zone transfer failed for zonetransfer.me against lisa.ns.cloudflare.com: dns: bad xfr rcode: 1
[-] Zone transfer failed for example.com against 1.1.1.1: dns: bad xfr rcode: 5

$ go run main.go -d zonetransfer.me@hal.ns.cloudflare.com
[*] Using nameserver: 10.0.0.1
[*] Attempting zone transfer for 1 domains
[-] Zone transfer failed for zonetransfer.me against hal.ns.cloudflare.com: dns: bad xfr rcode: 1
```