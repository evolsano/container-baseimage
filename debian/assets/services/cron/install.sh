#!/bin/bash -e

chmod 600 /etc/crontab

# fix https://github.com/phusion/baseimage-docker/issues/345
sed -i 's/^\s*session\s\+required\s\+pam_loginuid.so/# &/' /etc/pam.d/cron

# remove useless cron entries.
# checks for lost+found and scans for mtab.
rm -f /etc/cron.daily/standard /etc/cron.daily/upstart /etc/cron.daily/dpkg /etc/cron.daily/password /etc/cron.weekly/fstrim
