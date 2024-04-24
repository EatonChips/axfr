package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/spf13/pflag"
)

var (
	nameserver  string
	domainsFile string
	domains     []string

	verbose bool

	outputFormats      string
	outputJsonFileName string
	outputCsvFileName  string

	err error
)

func main() {
	pflag.StringVarP(&nameserver, "nameserver", "n", "", "DNS Server for resolving domain name servers")
	pflag.StringArrayVarP(&domains, "domains", "d", []string{}, "Domain names to transfer")
	pflag.StringVarP(&domainsFile, "file", "f", "", "File containing domain names to transfer")
	pflag.StringVarP(&outputFormats, "output", "o", "", "Output format (json,csv)")
	pflag.StringVarP(&outputJsonFileName, "json", "j", "", "Output file for json format")
	pflag.StringVarP(&outputCsvFileName, "csv", "c", "", "Output file for csv format")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	pflag.Parse()

	// Setup output files
	var outputJsonFile *os.File
	var outputCsvFile *os.File

	if outputFormats != "" {
		if strings.Contains(outputFormats, "json") && outputJsonFileName == "" {
			outputJsonFileName = fmt.Sprintf("axfr-%s.json", time.Now().Format("20060102-150405"))
		}

		if strings.Contains(outputFormats, "csv") && outputCsvFileName == "" {
			outputCsvFileName = fmt.Sprintf("axfr-%s.csv", time.Now().Format("20060102-150405"))
		}

		if strings.Contains(outputFormats, "json") {
			logInfo(fmt.Sprintf("Using json output file: %s", outputJsonFileName))
			outputJsonFile, err = os.Create(outputJsonFileName)
			if err != nil {
				logError(err.Error())
				return
			}
			defer outputJsonFile.Close()
		}

		if strings.Contains(outputFormats, "csv") {
			logInfo(fmt.Sprintf("Using csv output file: %s", outputCsvFileName))
			outputCsvFile, err = os.Create(outputCsvFileName)
			if err != nil {
				logError(err.Error())
				return
			}
			defer outputCsvFile.Close()

			outputCsvFile.WriteString("Domain,Nameserver,Name,Type,TTL,Value\n")
		}
	}

	// Append domains from file to domains array
	if domainsFile != "" {
		logInfo("Reading domains from file")
		file, err := os.Open(domainsFile)
		if err != nil {
			logError(err.Error())
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			logError(err.Error())
			return
		}

		fileDomains := strings.Split(string(content), "\n")
		domains = append(domains, fileDomains...)
	}

	if len(domains) == 0 {
		logError("No domains specified")
		pflag.Usage()
		return
	}

	if nameserver == "" {
		nameserver = getSystemNameserver()

		if nameserver == "" {
			logError("Failed to get system nameserver, specify with -n")
			return
		}
	}
	logInfo(fmt.Sprintf("Using nameserver: %s", nameserver))

	logInfo(fmt.Sprintf("Attempting zone transfer for %d domains", len(domains)))

	zoneTransferResults := []ZoneTransferResult{}

	for _, domain := range domains {
		if domain == "" {
			continue
		}

		nameservers := []string{}
		if strings.Contains(domain, "@") {
			splitDomain := strings.Split(domain, "@")
			domain = splitDomain[0]
			nameservers = append(nameservers, splitDomain[1])
		} else {
			nameservers, err = getNameservers(domain)
			if err != nil {
				logError(fmt.Sprintf("Failed to get nameservers for %s: %s", domain, err.Error()))
				continue
			}
		}

		if verbose {
			logInfo(fmt.Sprintf("Nameservers for %s: %s", domain, strings.Join(nameservers, ", ")))
		}

		for _, ns := range nameservers {
			if verbose {
				logInfo(fmt.Sprintf("Performing zone transfer for %s against %s", domain, ns))
			}

			zoneTransferResult, err := zoneTransfer(domain, ns)
			zoneTransferResults = append(zoneTransferResults, zoneTransferResult)
			if err != nil {
				logUnsuccess(fmt.Sprintf("Zone transfer failed for %s against %s: %s", domain, ns, err.Error()))
				continue
			}

			logSuccess(fmt.Sprintf("Zone transfer successful for %s against %s, identified %d records", domain, ns, len(zoneTransferResult.Records)))

			if verbose {
				header := fmt.Sprintf("=== %s@%s =======================================", domain, ns)
				logSuccess(header)

				for _, record := range zoneTransferResult.Records {
					logSuccess(fmt.Sprintf("Name: %s, Type: %s, TTL: %d, Value: %s", record.Name, record.Type, record.TTL, record.Value))
				}

				logSuccess(strings.Repeat("=", len(header)))
			}

			if outputCsvFile != nil {
				for _, record := range zoneTransferResult.Records {
					outputCsvFile.WriteString(fmt.Sprintf("%s,%s,%s,%s,%d,%s\n", domain, ns, record.Name, record.Type, record.TTL, record.Value))
				}
			}
		}
	}

	if outputJsonFile != nil {
		resultJSON, err := json.MarshalIndent(zoneTransferResults, "", "  ")
		if err != nil {
			logError(err.Error())
		}
		outputJsonFile.Write(resultJSON)
	}

}

func getNameservers(domain string) ([]string, error) {
	var nameservers []string

	c := new(dns.Client)

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeNS)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, nameserver+":53")
	if err != nil {
		return []string{""}, err
	}

	if r.Rcode != dns.RcodeSuccess {
		return []string{""}, errors.New("error: rcode != dns.RcodeSuccess")
	}

	for _, ans := range r.Answer {
		if ns, ok := ans.(*dns.NS); ok {
			nameservers = append(nameservers, strings.TrimSuffix(ns.Ns, "."))
		}
	}

	return nameservers, nil
}

type ZoneTransferResult struct {
	Domain     string                      `json:"domain"`
	Nameserver string                      `json:"nameserver"`
	Records    []ZoneTransferResultRecords `json:"records"`
}

type ZoneTransferResultRecords struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	TTL   uint32 `json:"ttl"`
	Value string `json:"value"`
}

func zoneTransfer(domain string, nameserver string) (ZoneTransferResult, error) {
	result := ZoneTransferResult{
		Domain:     domain,
		Nameserver: nameserver,
		Records:    []ZoneTransferResultRecords{},
	}

	m := new(dns.Msg)
	m.SetAxfr(dns.Fqdn(domain))
	t := new(dns.Transfer)

	ch, err := t.In(m, nameserver+":53")
	if err != nil {
		return result, err
	}

	for envelope := range ch {
		if envelope.Error != nil {
			return result, envelope.Error
		}

		resultRecords := []ZoneTransferResultRecords{}
		for _, rr := range envelope.RR {
			// Easier to pull the value from a string than to switch on every single type
			recordValue := fmt.Sprintf("%T\t%[1]s\n", rr)
			recordValue = strings.Split(recordValue, "\t")[5]
			recordValue = strings.TrimSuffix(recordValue, "\n")

			resultRecords = append(resultRecords, ZoneTransferResultRecords{
				Type:  dns.TypeToString[rr.Header().Rrtype],
				Name:  rr.Header().Name,
				TTL:   rr.Header().Ttl,
				Value: recordValue,
			})
		}

		result.Records = resultRecords
	}

	return result, nil
}

// TODO: One day, maybe we can get the Windows DNS server. One day.
func getSystemNameserver() string {
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return ""
	}

	if len(config.Servers) > 0 {
		return config.Servers[0]
	}

	return ""
}

func logInfo(msg string) {
	blue := color.New(color.FgBlue).SprintFunc()
	fmt.Printf("%s %s\n", blue("[*]"), msg)
}

func logError(msg string) {
	red := color.New(color.FgRed).SprintFunc()
	fmt.Printf("%s %s\n", red("[!]"), msg)
}

func logSuccess(msg string) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s %s\n", green("[+]"), msg)
}

func logUnsuccess(msg string) {
	red := color.New(color.FgRed).SprintFunc()
	fmt.Printf("%s %s\n", red("[-]"), msg)
}
