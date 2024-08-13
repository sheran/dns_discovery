package main

import (
	"fmt"
	"net"
)

func host2ip(hostList []string) map[string][]string {
	database := make(map[string][]string)
	for _, host := range hostList {
		if err := lookupHost(host, &database); err != nil {
			if (err.(*net.DNSError)).IsNotFound {
				fmt.Printf("%s,0.0.0.0\n", host)
			} else {
				fmt.Printf("%s,%s", host, err.Error())
			}
		}
	}
	sorted := sortByIP(database)
	return sorted
}

func lookupHost(host string, ipDb *map[string][]string) error {
	ips, err := net.LookupIP(host)
	if err != nil {
		if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
			return err
		}
		return fmt.Errorf("lookup failed: %v", err)
	}
	ipList := make([]string, 0)
	for _, ip := range ips {
		if ip.To4() != nil {
			ipList = append(ipList, ip.String())
		}
	}
	(*ipDb)[host] = ipList
	return nil
}

func sortByIP(ipDb map[string][]string) map[string][]string {
	sorted := make(map[string][]string)
	for host, ips := range ipDb {
		for _, ip := range ips {
			if _, ok := sorted[ip]; !ok {
				sorted[ip] = make([]string, 0)
			}
			sorted[ip] = append(sorted[ip], host)
		}
	}
	return sorted
}
