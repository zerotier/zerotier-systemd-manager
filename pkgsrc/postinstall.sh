#!/bin/sh

set -xeu

test -f /var/run/zerotier-one.service && mv /var/run/zerotier-one.service /lib/systemd/system
systemctl daemon-reload
systemctl enable zerotier-systemd-manager.timer
systemctl start zerotier-systemd-manager.timer
