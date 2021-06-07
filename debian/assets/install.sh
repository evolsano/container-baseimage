#!/bin/bash -e

# install required packages
packages-index-update
packages-install-clean bash-completion locales eatmydata

# set locale
echo "en_US.UTF-8 UTF-8" >> /etc/locale.gen
locale-gen en_US.UTF-8 2>&1 | container-logger info
update-locale LANG=en_US.UTF-8 LC_CTYPE=en_US.UTF-8

# add container-baseimage bash completion
container-baseimage completion bash > /usr/share/bash-completion/completions/container-baseimage
echo ". /etc/profile.d/bash_completion.sh" >> /root/.bashrc

# clean
rm -rf /tmp/* /var/tmp/*
