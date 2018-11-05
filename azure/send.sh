#!/usr/bin/env bash

#!/bin/bash

# Note: depends on sshpass to pass password to scp:
# https://gist.github.com/arunoda/7790979

if [ "$#" -lt 2 ]; then
    echo "Illegal number of parameters"
    echo "usage:"
    echo "[username, password, opts, command]"
    exit
fi


USER=$1     # username for scp
PASS=$2     # password for scp
OPTS=$3
COMMAND=$4

ExecOnRepo() {
    VMIP=$1        # ip of the vm to scp to
    COMMAND=$2     # command to run
    echo -e "Sending: ${USER}@${VMIP} ${COMMAND}"
    sshpass -p ${PASS} ssh ${USER}@${VMIP} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no "cd ~/P1-e5w9a-b2v9a; ${COMMAND}" &
}

IPS=(
"13.77.181.113"
"13.66.184.152"
"13.77.145.38"
"13.77.158.176"
"13.77.157.8"
"13.77.156.153"
"13.77.156.10"
)

LOCAL_IPS=(
"10.0.0.4"
"10.0.0.5"
"10.0.0.6"
"10.0.0.8"
"10.0.0.9"
"10.0.0.10"
"10.0.0.11"
)

UpdateConfig() {
    for i in ${IPS[*]}; do
        ExecOnRepo ${i} "sed -i \
        's/\\("'"'"OutgoingMinersIP"'"'" :\\).*$/\\1 "'"'"${i}"'"'",/g' testfiles/config_good.json"
    done
    wait

    COUNTER=0
    for i in ${IPS[*]}; do
        ExecOnRepo ${i} "sed -i \
        's/\\("'"'"IncomingMinersAddr"'"'" :\\).*$/\\1 "'"'"${LOCAL_IPS[${COUNTER}]}:5050"'"'",/g' testfiles/config_good.json"
        COUNTER=$((COUNTER + 1))
    done
    wait

    COUNTER=0
    for i in ${IPS[*]}; do
        ExecOnRepo ${i} "sed -i \
        's/\\("'"'"IncomingClientsAddr"'"'" :\\).*$/\\1 "'"'"${LOCAL_IPS[${COUNTER}]}:9090"'"'"/g' testfiles/config_good.json"
        COUNTER=$((COUNTER + 1))
    done
    wait

    for i in ${IPS[*]}; do
        ExecOnRepo ${i} "sed -i \
        's/\\("'"'"PowPerOpBlock"'"'" :\\).*$/\\1 7,/g' testfiles/config_good.json"
    done
    wait

    COUNTER=0
    for i in ${IPS[*]}; do
        ExecOnRepo ${i} "sed -i \
        's/\\("'"'"MinerID"'"'" :\\).*$/\\1 "'"'"Miner_${COUNTER}"'"'",/g' testfiles/config_good.json"
        COUNTER=$((COUNTER + 1))
    done
    wait
}

PEERS=(
'[]'
'["'"'"'""13.77.181.113:5050""'"'"'"]'
'["'"'"'""13.66.184.152:5050""'"'"'"]'
'["'"'"'""13.77.145.38:5050""'"'"'"]'
'["'"'"'""13.77.158.176:5050""'"'"'"]'
'["'"'"'""13.77.157.8:5050""'"'"'"]'
'["'"'"'""13.77.156.153:5050""'"'"'"]'
'["'"'"'""13.77.156.10:5050""'"'"'"]')

AddPeers() {
    COUNTER=0
    for i in ${IPS[*]}; do
        ExecOnRepo ${i} "sed -i \
        's/\\( "'"'"PeerMinersAddrs"'"'" :\\).*$/\\1 ${PEERS[${COUNTER}]},/g' testfiles/config_good.json"
        COUNTER=$((COUNTER + 1))
    done
    wait
}

StartMiners() {
    for i in ${IPS[*]}; do
        tmux new-window "sshpass -p '${PASS}' ssh ${USER}@${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'cd P1-e5w9a-b2v9a; \
        /usr/lib/go-1.9/bin/go run ./miner.go testfiles/config_good.json'"
        sleep 3
    done
    tmux attach
    wait
}

# default behavior, assume that we want all
if [[ ${OPTS} == "" ]]; then
    UpdateConfig
    AddPeers
    StartMiners
fi

if [[ ${OPTS} == "-c" || ${OPTS} == "-C" ]]; then
    for i in ${IPS[*]}; do
        ExecOnRepo ${i} ${COMMAND} &
    done
    wait
fi


if [[ ${OPTS} == "-s" || ${OPTS} == "-S" ]]; then
    StartMiners
fi