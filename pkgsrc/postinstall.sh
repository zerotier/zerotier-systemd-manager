#!/bin/sh

set -xeu

systemctl daemon-reload
systemctl enable zerotier-systemd-manager.timer
systemctl start zerotier-systemd-manager.timer
