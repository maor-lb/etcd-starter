// Copyright 2015 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package discovery

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/coreos/etcd-starter/Godeps/_workspace/src/github.com/coreos/etcd/pkg/types"
)

var (
	// indirection for testing
	lookupSRV = net.LookupSRV
)

// TODO(barakmich): Currently ignores priority and weight (as they don't make as much sense for a bootstrap)
// Also doesn't do any lookups for the token (though it could)
// Also sees each entry as a separate instance.
func SRVGetCluster(name, dns string, defaultToken string, apurls types.URLs) (string, string, error) {
	stringParts := make([]string, 0)
	tempName := int(0)
	tcpAPUrls := make([]string, 0)

	// First, resolve the apurls
	for _, url := range apurls {
		tcpAddr, err := net.ResolveTCPAddr("tcp", url.Host)
		if err != nil {
			log.Printf("discovery: Couldn't resolve host %s during SRV discovery", url.Host)
			return "", "", err
		}
		tcpAPUrls = append(tcpAPUrls, tcpAddr.String())
	}

	updateNodeMap := func(service, prefix string) error {
		_, addrs, err := lookupSRV(service, "tcp", dns)
		if err != nil {
			return err
		}
		for _, srv := range addrs {
			host := net.JoinHostPort(srv.Target, fmt.Sprintf("%d", srv.Port))
			tcpAddr, err := net.ResolveTCPAddr("tcp", host)
			if err != nil {
				log.Printf("discovery: Couldn't resolve host %s during SRV discovery", host)
				continue
			}
			n := ""
			for _, url := range tcpAPUrls {
				if url == tcpAddr.String() {
					n = name
				}
			}
			if n == "" {
				n = fmt.Sprintf("%d", tempName)
				tempName += 1
			}
			stringParts = append(stringParts, fmt.Sprintf("%s=%s%s", n, prefix, tcpAddr.String()))
			log.Printf("discovery: Got bootstrap from DNS for %s at host %s to %s%s", service, host, prefix, tcpAddr.String())
		}
		return nil
	}

	failCount := 0
	err := updateNodeMap("etcd-server-ssl", "https://")
	if err != nil {
		log.Printf("discovery: Error querying DNS SRV records for _etcd-server-ssl %s", err)
		failCount += 1
	}
	err = updateNodeMap("etcd-server", "http://")
	if err != nil {
		log.Printf("discovery: Error querying DNS SRV records for _etcd-server %s", err)
		failCount += 1
	}
	if failCount == 2 {
		log.Printf("discovery: SRV discovery failed: too many errors querying DNS SRV records")
		return "", "", err
	}
	return strings.Join(stringParts, ","), defaultToken, nil
}
