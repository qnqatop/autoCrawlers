// Copyright 2018 Adam Tauber
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"bufio"
	"context"
	"embed"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/gocolly/colly/v2"
)

type Balancer struct {
	Proxies []string
}

//go:embed proxies.txt
var proxiesFile embed.FS

func NewBalancer() *Balancer {
	return &Balancer{
		Proxies: make([]string, 0),
	}
}

func (b *Balancer) Load() (int, error) {
	file, err := proxiesFile.Open("proxies.txt")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		b.Proxies = append(b.Proxies, scanner.Text())
	}

	return len(b.Proxies), scanner.Err()
}

type roundRobinSwitcher struct {
	proxyURLs []*url.URL
	index     uint32
}

// nolint
func (r *roundRobinSwitcher) GetProxy(pr *http.Request) (*url.URL, error) {
	u := r.proxyURLs[r.index%uint32(len(r.proxyURLs))]
	atomic.AddUint32(&r.index, 1)
	ctx := context.WithValue(pr.Context(), colly.ProxyURLKey, u.String())
	*pr = *pr.WithContext(ctx)
	return u, nil
}

// RoundRobinProxySwitcher creates a proxy switcher function which rotates
// ProxyURLs on every request.
// The proxy type is determined by the URL scheme. "http", "https"
// and "socks5" are supported. If the scheme is empty,
// "http" is assumed.
func (b *Balancer) RoundRobinProxySwitcher() (colly.ProxyFunc, error) {
	urls := make([]*url.URL, len(b.Proxies))
	for i, u := range b.Proxies {
		parsedU, err := url.Parse(u)
		if err != nil {
			return nil, err
		}
		urls[i] = parsedU
	}
	return (&roundRobinSwitcher{urls, 0}).GetProxy, nil
}
