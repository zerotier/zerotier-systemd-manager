package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/zerotier/zerotier-systemd-manager/service"
)

//go:embed template.network
var networkTemplate string

type templateScaffold struct {
	Interface   string
	NetworkName string
	DNS         string
	DNSSearch   string
}

type serviceAPIClient struct {
	apiKey string
	client *http.Client
}

func NewServiceAPI() (*serviceAPIClient, error) {
	content, err := ioutil.ReadFile("/var/lib/zerotier-one/authtoken.secret")
	if err != nil {
		return nil, err
	}

	return &serviceAPIClient{apiKey: strings.TrimSpace(string(content)), client: &http.Client{}}, nil
}

func errExit(msg interface{}) {
	fmt.Fprintf(os.Stderr, "%v\n", msg)
	os.Exit(1)
}

// Do initiates a client transaction.
func (c *serviceAPIClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Add("X-ZT1-Auth", c.apiKey)
	return c.client.Do(req)
}

func main() {
	if os.Geteuid() != 0 {
		errExit("You need to be root to run this program")
	}

	if runtime.GOOS != "linux" {
		errExit("This tool is only needed (and useful) on linux")
	}

	t, err := template.New("network").Parse(networkTemplate)
	if err != nil {
		errExit("your template is busted; get a different version or stop modifying the source code :)")
	}

	sAPI, err := NewServiceAPI()
	if err != nil {
		errExit(err)
	}

	client, err := service.NewClient("http://localhost:9993", service.WithHTTPClient(sAPI))
	if err != nil {
		errExit(err)
	}

	resp, err := client.GetNetworks(context.Background())
	if err != nil {
		errExit(err)
	}

	networks, err := service.ParseGetNetworksResponse(resp)
	if err != nil {
		errExit(err)
	}

	for _, network := range *networks.JSON200 {
		if network.Dns != nil && len(*network.Dns.Servers) != 0 {
			fn := fmt.Sprintf("/etc/systemd/network/99-%s.network", *network.PortDeviceName)
			fmt.Printf("Generating %q\n", fn)

			var search string
			if network.Dns.Domain != nil {
				search = *network.Dns.Domain
			}

			out := templateScaffold{
				Interface:   *network.PortDeviceName,
				NetworkName: *network.Name,
				DNS:         strings.Join(*network.Dns.Servers, ","),
				DNSSearch:   search,
			}

			buf := bytes.NewBuffer(nil)

			if err := t.Execute(buf, out); err != nil {
				errExit(fmt.Errorf("%q: %w", fn, err))
			}

			if _, err := os.Stat(fn); err == nil {
				content, err := ioutil.ReadFile(fn)
				if err != nil {
					errExit(fmt.Errorf("In %v: %w", fn, err))
				}

				if bytes.Equal(content, buf.Bytes()) {
					fmt.Printf("%q hasn't changed; skipping\n", fn)
					continue
				}
			}

			f, err := os.Create(fn)
			if err != nil {
				errExit(fmt.Errorf("%q: %w", fn, err))
			}

			if _, err := f.Write(buf.Bytes()); err != nil {
				errExit(err)
			}
		}
	}
}
