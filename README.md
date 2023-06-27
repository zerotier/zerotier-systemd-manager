# Get ZeroTier playing nice with systemd-networkd and resolvectl

This is a small tool to enable the systemd-networkd service as well as resolvectl to enable per-interface DNS settings. We take this directly from zerotier-one (on your machine) metadata. This service does not reach out to the internet on its own.

The result is per-interface DNS settings, which is especially nice when you are using [zeronsd](https://github.com/zerotier/zeronsd) with multiple networks.

## Usage

[Check out our releases for debian and redhat packages that automate this on a variety of platforms](https://github.com/zerotier/zerotier-systemd-manager/releases).

### Installing From Source

Compile it with [golang 1.16 or later](https://golang.org):

```bash
# be outside of gopath when you do this
go get github.com/zerotier/zerotier-systemd-manager
```

Ensure `systemd-networkd` is properly configured and `resolvectl` works as intended.

Finally, run the tool as `root`: `zerotier-systemd-manager`. If you have interfaces with DNS assignments in ZeroTier, it will populate files in `/etc/systemd/network`. No DNS assignment, no file. Unless you have passed `-auto-restart=false`, it will restart `systemd-networkd` for you if things have changed.

If you have a DNS-over-TLS configuration provided by zeronsd (v0.4.0 or later), you can enable using it by providing `-dns-over-tls=true` in the supervisor (a systemd timer in the default case). You will have to hand-edit this in for now.

If you want to enable multicast DNS / bonjour / mDNS you can enable it by providing `-multicast-dns`.

Finally, if you have left a DNS-controlled network it will try to remove the old files if `-reconcile=true` is set (the default). This way you can stuff it in cron and not think about it too much.

Enjoy!

## Author

Erik Hollensbe <github@hollensbe.org>

## License

BSD 3-Clause

## Releasing
This repo uses [goreleaser](https://goreleaser.com/quick-start/). 

