#!/usr/bin/env bash

# get deployment ip addresses
# ONLY TO GET HOSTS ADD THEM TO LST
# az vmss list-instance-public-ips --name TestSet --resource-group myresource | grep -i \"ipAddress\" | awk '{ print $2}' | sed 's/,//'

# FILL IN THE LIST OF HOSTS
LST=(
"13.77.181.113"
"13.66.184.152"
"13.77.145.38"
"13.77.158.176"
"13.77.157.8"
"13.77.156.153"
"13.77.156.10"
)

USER="b2v9a"     # username for scp
PASS="azure1234567890!"     # password for scp

# assuming you've setup the ssh key correctly
for i in ${LST[*]}; do
    sshpass -p ${PASS} ssh ${USER}@${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'sudo add-apt-repository ppa:gophers/archive; sudo apt-get update; sudo apt-get -y install golang-1.9-go' &
done
wait

for i in ${LST[*]}; do
    sshpass -p ${PASS} ssh ${USER}@${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'tee -a ~/.ssh/id_rsa << END
!!!!!!!!!!!
END' &
done
wait

for i in ${LST[*]}; do
    sshpass -p ${PASS} ssh ${USER}@${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'chmod 700 ~/.ssh/id_rsa' &
done
wait

for i in ${LST[*]}; do
    sshpass -p ${PASS} ssh ${USER}@${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'tee -a ~/.ssh/config << END
Host github.ugrad.cs.ubc.ca
  IdentityFile ~/.ssh/id_rsa
END' &
done
wait

for i in ${LST[*]}; do
    sshpass -p ${PASS} ssh ${USER}@${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'chmod 700 ~/.ssh/config' &
done
wait

for i in ${LST[*]}; do
    sshpass -p ${PASS} ssh ${USER}@${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'sudo apt-get -y install git' &
done
wait

for i in ${LST[*]}; do
    sshpass -p ${PASS} ssh ${USER}@${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'GIT_SSH_COMMAND="ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no" git clone git@github.ugrad.cs.ubc.ca:CPSC416-2018W-T1/P1-e5w9a-b2v9a.git' &
done
wait

for i in ${LST[*]}; do
    sshpass -p ${PASS} ssh ${USER}@${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'cd P1-e5w9a-b2v9a; git reset --hard; GIT_SSH_COMMAND="ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no" git pull origin master' &
done
wait

for i in ${LST[*]}; do
    sshpass -p ${PASS} ssh ${USER}@${i} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no 'cd P1-e5w9a-b2v9a; chmod a+x install; PATH=/usr/lib/go-1.9/bin/:\\$PATH ./install' &
done
wait