#!/bin/sh
set -e

mkdir -p /data/wallpapers /data/icons
chown -R naviodeck:naviodeck /data

exec su-exec naviodeck /app/naviodeck "$@"
