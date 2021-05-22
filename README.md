# Get ZeroTier playing nice with systemd-networkd and resolvectl

This is a small tool to enable the systemd-networkd service as well as resolvectl to enable per-interface DNS settings. We take this directly from zerotier-one (on your machine) metadata. This service does not reach out to the internet on its own.

The result is per-interface DNS settings, which is especially nice when you are using [zeronsd](https://github.com/zerotier/zeronsd) with multiple networks.

## Usage

Compile it with [golang](https://golang.org):

```bash
# be outside of gopath when you do this
go get github.com/zerotier/zerotier-systemd-manager
```

Install our [slightly modified zerotier-one.service file](contrib/zerotier-one.service) in `/usr/lib/systemd/system` on Ubuntu, but this location may be different for other operating systems. This will make `zerotier-one` depend on `systemd-networkd`.

Ensure `systemd-networkd` is properly configured and `resolvectl` works as intended.

Finally, run the tool as `root`: `zerotier-systemd-manager`. If you have interfaces with DNS assignments in ZeroTier, it will populate files in `/etc/systemd/network`. No DNS assignment, no file. Unless you have passed `-auto-restart=false`, it will restart `systemd-networkd` for you if things have changed.

There is no detection of left networks, to avoid destroying files you care about. Remove these files manually. For better or worse, they are set up in a way that creates no interference, so as long as you don't get a similar interface name (unlikely), you're on solid ground.

## Author

Erik Hollensbe <github@hollensbe.org>

## License

BSD 3-Clause
