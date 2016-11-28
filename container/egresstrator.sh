#!/bin/bash
# Adhere to: https://github.com/progrium/bashstyle
set -euf -o pipefail

usage() {
    echo $"Usage: $0 {set-egress|clear-egress}"
    exit 1
}

set_egress() {
    echo "set-egress"
    /usr/bin/consul-template \
    -template "iptables.ctmpl:/iptables:/egresstrator.sh run" \
    -wait 1s:2s \
    -once
}

clear_egress() {
    echo "clear-egress"
    iptables -F
}

run() {
    sleep 1
    cat /iptables | iptables-restore
    iptables-save
    exit 0
}

if [[ $# -ne 1 ]] ; then
    usage
    exit 1
fi

case $1 in
    set-egress)
        set_egress
        ;;
    clear-egress)
        clear_egress
        ;;
    run)
        run
        ;;
    *)
        usage
    ;;
esac
