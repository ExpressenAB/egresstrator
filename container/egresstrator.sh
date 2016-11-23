#!/bin/bash
set -euf -o pipefail

if [[ $# -ne 1 ]] ; then
    echo "Usage: ${0} <gw>"
    exit 1
fi

GW=${1}
sleep 3 # We will never regret this!
/sbin/ip a
/sbin/ip route change default via ${GW}
