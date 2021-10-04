package main

import (
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/zerotier/zerotier-systemd-manager/service"
)

//go:embed template.network
var networkTemplate string

const (
	magicComment = "--- Managed by zerotier-systemd-manager. Do not remove this comment. ---"
	networkDir   = "/etc/systemd/network"
	ipv4bits     = net.IPv4len * 8
)

// parameter list for multiple template operations
type templateScaffold struct {
	Interface    string
	NetworkName  string
	DNS          []string
	DNSSearch    string
	MagicComment string
}

// wrapped openapi client. should probably be replaced with a code generator in
// a separate repository as it's always out of date.
type serviceAPIClient struct {
	apiKey string
	client *http.Client
}

func newServiceAPI() (*serviceAPIClient, error) {
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

// Do initiates a client transaction. 99% of what the wrapped client does is inject the right header.
func (c *serviceAPIClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Add("X-ZT1-Auth", c.apiKey)
	return c.client.Do(req)
}

func main() {
	// two flags for the CLI auto-restart and reconcile are defaulted to true, so you rarely need them.
	autoRestartFlag := flag.Bool("auto-restart", true, "Automatically restart systemd-resolved when things change")
	reconcileFlag := flag.Bool("reconcile", true, "Automatically remove left networks from systemd-networkd configuration")
	flag.Parse()

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

	/*
		 	 this bit is fundamentally a set difference of the networks zerotier knows
			 about, and systemd-resolved knows about. If reconcile is true, this is
			 corrected. If any corrections are made, or any networks added, and
			 auto-restart is true, then systemd-resolved is reloaded near the end.
	*/

	sAPI, err := newServiceAPI()
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

	dir, err := os.ReadDir(networkDir)
	if err != nil {
		errExit(err)
	}

	found := map[string]struct{}{}

	for _, item := range dir {
		if item.Type().IsRegular() && strings.HasSuffix(item.Name(), ".network") {
			content, err := ioutil.ReadFile(filepath.Join(networkDir, item.Name()))
			if err != nil {
				errExit(err)
			}

			if bytes.Contains(content, []byte(magicComment)) {
				found[item.Name()] = struct{}{}
			}
		}
	}

	var changed bool

	for _, network := range *networks.JSON200 {
		if network.Dns != nil && len(*network.Dns.Servers) != 0 {
			fn := fmt.Sprintf("/etc/systemd/network/99-%s.network", *network.PortDeviceName)

			delete(found, path.Base(fn))

			search := map[string]struct{}{}

			if network.Dns.Domain != nil {
				search[*network.Dns.Domain] = struct{}{}
			}

			// This calculates in-addr.arpa and ip6.arpa search domains by calculating them from the IP assignments.
			if network.AssignedAddresses != nil && len(*network.AssignedAddresses) > 0 {
				for _, addr := range *network.AssignedAddresses {
					ip, ipnet, err := net.ParseCIDR(addr)
					if err != nil {
						errExit(fmt.Sprintf("Could not parse CIDR %q: %v", addr, err))
					}

					used, total := ipnet.Mask.Size()
					bits := int(math.Ceil(float64(total) / float64(used)))

					octets := make([]byte, bits)
					if total == ipv4bits {
						ip = ip.To4()
					}

					for i := 0; i < bits; i++ {
						octets[i] = ip[i]
					}

					searchLine := "~"
					for i := len(octets) - 1; i >= 0; i-- {
						if total > ipv4bits {
							searchLine += fmt.Sprintf("%x.", (octets[i] & 0xf))
							searchLine += fmt.Sprintf("%x.", ((octets[i] >> 4) & 0xf))
						} else {
							searchLine += fmt.Sprintf("%d.", octets[i])
						}
					}

					if total == ipv4bits {
						searchLine += "in-addr.arpa"
					} else {
						searchLine += "ip6.arpa"
					}

					search[searchLine] = struct{}{}
				}
			}

			searchkeys := []string{}

			for key := range search {
				searchkeys = append(searchkeys, key)
			}

			sort.Strings(searchkeys)

			out := templateScaffold{
				Interface:    *network.PortDeviceName,
				NetworkName:  *network.Name,
				DNS:          *network.Dns.Servers,
				DNSSearch:    strings.Join(searchkeys, " "),
				MagicComment: magicComment,
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
					fmt.Fprintf(os.Stderr, "%q hasn't changed; skipping\n", fn)
					continue
				}
			}

			fmt.Printf("Generating %q\n", fn)
			f, err := os.Create(fn)
			if err != nil {
				errExit(fmt.Errorf("%q: %w", fn, err))
			}

			if _, err := f.Write(buf.Bytes()); err != nil {
				errExit(err)
			}

			f.Close()

			changed = true
		}
	}

	if len(found) > 0 && *reconcileFlag {
		fmt.Println("Found unused networks, reconciling...")

		for fn := range found {
			fmt.Printf("Removing %q\n", fn)

			if err := os.Remove(filepath.Join(networkDir, fn)); err != nil {
				errExit(fmt.Errorf("While removing %q: %w", fn, err))
			}
		}
	}

	if (changed || len(found) > 0) && *autoRestartFlag {
		fmt.Println("Files changed; reloading systemd-networkd...")

		if err := exec.Command("networkctl", "reload").Run(); err != nil {
			errExit(fmt.Errorf("While reloading systemd-networkd: %v", err))
		}
	}
}
