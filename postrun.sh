#!/bin/sh

set -xeu

mv /var/run/zerotier-one.service /lib/systemd/system
systemctl daemon-reload
systemctl enable zerotier-systemd-manager.timer
systemctl start zerotier-systemd-manager.timer
