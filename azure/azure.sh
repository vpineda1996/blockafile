#!/usr/bin/env bash

# get deployment ip addresses
# ONLY TO GET HOSTS ADD THEM TO LST
# az vmss list-instance-public-ips --name TestSet --resource-group myresource | grep -i \"ipAddress\" | awk '{ print $2}' | sed 's/,//'

# FILL IN THE LIST OF HOSTS
LST=("13.66.175.108"
"13.66.205.72"
"52.183.9.120"
"52.183.8.178"
"52.183.14.52"
"52.183.14.47"
"52.183.13.104"
"52.183.0.59"
"52.183.10.214"
"52.183.10.16"
)

# assuming you've setup the ssh key correctly
for i in ${LST[*]}; do
    ssh $i -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'sudo add-apt-repository ppa:gophers/archive; sudo apt-get update; sudo apt-get -y install golang-1.10-go' &
done
wait

for i in ${LST[*]}; do
    ssh $i -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'tee -a ~/.ssh/id_rsa << END
!!!!!!!!!!!!!!!!!!! ADDD KEY !!!!!!!!!!!!!!!!
END' &
done
wait

for i in ${LST[*]}; do
    ssh $i -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'chmod 700 ~/.ssh/id_rsa' &
done
wait

for i in ${LST[*]}; do
    ssh $i -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'tee -a ~/.ssh/config << END
Host github.ugrad.cs.ubc.ca
  IdentityFile ~/.ssh/id_rsa
END' &
done
wait

for i in ${LST[*]}; do
    ssh $i -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'chmod 700 ~/.ssh/config' &
done
wait

for i in ${LST[*]}; do
    ssh $i -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'sudo apt-get -y install git' &
done
wait

for i in ${LST[*]}; do
    ssh $i -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'GIT_SSH_COMMAND="ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no" git clone git@github.ugrad.cs.ubc.ca:CPSC416-2018W-T1/P1-e5w9a-b2v9a.git' &
done
wait

for i in ${LST[*]}; do
    ssh $i -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'cd P1-e5w9a-b2v9a; git reset --hard; GIT_SSH_COMMAND="ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no" git pull origin master' &
done
wait

for i in ${LST[*]}; do
    ssh $i -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'cd P1-e5w9a-b2v9a; chmod a+x install; PATH=/usr/lib/go-1.10/bin/:\\$PATH ./install' &
done
wait


#enter the string of peers here
for i in ${LST[*]}; do
    ssh ${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no "cd P1-e5w9a-b2v9a; sed -i \
    's/\\("'"'"OutgoingMinersIP"'"'" :\\).*$/\\1 "'"'"${i}"'"'",/g' testfiles/config_good.json" &
done
wait

for i in ${LST[*]}; do
    ssh ${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no "cd P1-e5w9a-b2v9a; sed -i \
    's/\\("'"'"IncomingMinersAddr"'"'" :\\).*$/\\1 "'"'":5050"'"'",/g' testfiles/config_good.json" &
done
wait

for i in ${LST[*]}; do
    ssh ${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no "cd P1-e5w9a-b2v9a; sed -i \
    's/\\("'"'"PowPerOpBlock"'"'" :\\).*$/\\1 8,/g' testfiles/config_good.json" &
done
wait

for i in ${LST[*]}; do
    ssh ${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no "cd P1-e5w9a-b2v9a; sed -i \
    's/\\("'"'"IncomingClientsAddr"'"'" :\\).*$/\\1 "'"'":9090"'"'"/g' testfiles/config_good.json" &
done
wait
COUNTER=0
for i in ${LST[*]}; do
    ssh ${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no "cd P1-e5w9a-b2v9a; sed -i \
    's/\\("'"'"MinerID"'"'" :\\).*$/\\1 "'"'"Miner_${COUNTER}"'"'",/g' testfiles/config_good.json" &
    COUNTER=$((COUNTER + 1))
done
wait

PEERS=('[]'
'["'"'"'""13.66.175.108:5050""'"'"'"]'
'["'"'"'""13.66.205.72:5050""'"'"'"]'
'["'"'"'""52.183.9.120:5050""'"'"'"]'
'["'"'"'""52.183.8.178:5050""'"'"'"]'
'["'"'"'""52.183.14.52:5050""'"'"'"]'
'["'"'"'""52.183.14.47:5050""'"'"'"]'
'["'"'"'""52.183.13.104:5050""'"'"'"]'
'["'"'"'""52.183.0.59:5050""'"'"'"]'
'["'"'"'""52.183.10.214:5050""'"'"'"]'
'["'"'"'""52.183.10.16:5050""'"'"'"]')
COUNTER=0
for i in ${LST[*]}; do
    ssh ${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no "cd P1-e5w9a-b2v9a; sed -i \
    's/\\( "'"'"PeerMinersAddrs"'"'" :\\).*$/\\1 ${PEERS[${COUNTER}]},/g' testfiles/config_good.json" &
    COUNTER=$((COUNTER + 1))
done
wait

for i in ${LST[*]}; do
    tmux new-window "ssh ${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'cd P1-e5w9a-b2v9a; \
    /usr/lib/go-1.10/bin/go run ./miner.go testfiles/config_good.json'"
    sleep 3
done
tmux attach
wait