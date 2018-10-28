#!/usr/bin/env bash

COMMANDS=("append.go" "cat.go" "head.go" "ls.go" "tail.go" "touch.go")
SERVER_IP=("localhost:9090" "localhost:9091")


# ARGS command idx, server IP idx, expected fail ... args
SendCommand() {
    touch .rfs
    echo localhost:$(( (${RANDOM} % 60000) + 1000 )) > .rfs
    echo ${SERVER_IP[$2]} >> .rfs
    command=${COMMANDS[$1]}

    echo "Running: go run $command $4 $5"
    go run ../${command} $4 $5
    if [ $? -gt 0 ]; then
        if [[ "$3" == "f" ]]; then
            echo Command returned exit code different than zero $?
        fi
    elif [[ "$3" == "t" ]]; then
        echo Expected command to fail but succeded
    fi
    rm .rfs
}

RunTest1 () {
    WAIT_TIME=5

    COMMAND_IDX=(5 0 0 3)
    NUMBER_OF_STEPS=${#COMMAND_IDX[@]}
    SERVER_IP_IDX=(0 1)
    EXPECTED_FAIL=("f" "f")
    ARG_1=("simple.txt" "")
    ARG_2=("" "")
    for i in `seq 0 $((${NUMBER_OF_STEPS} - 1))`;
    do
        SendCommand ${COMMAND_IDX[i]} ${SERVER_IP_IDX[i]} ${EXPECTED_FAIL[i]} ${ARG_1[i]} ${ARG_2[i]}
        sleep ${WAIT_TIME}s
    done
}

#RunTest1

RunTest2 () {
    WAIT_TIME=5

    COMMAND_IDX=(5 0 0 3)
    NUMBER_OF_STEPS=${#COMMAND_IDX[@]}
    SERVER_IP_IDX=(0 0 0 1)
    EXPECTED_FAIL=("f" "f" "f" "f")
    ARG_1=("simple.txt" "simple.txt" "simple.txt" "")
    ARG_2=("" "my_oh_god_no" "_no_no_no" "")
    for i in `seq 0 $((${NUMBER_OF_STEPS} - 1))`;
    do
        SendCommand ${COMMAND_IDX[i]} ${SERVER_IP_IDX[i]} ${EXPECTED_FAIL[i]} ${ARG_1[i]} ${ARG_2[i]}
        sleep ${WAIT_TIME}s
    done
}

#RunTest2

RunTest3 () {
    WAIT_TIME=5

    COMMAND_IDX=(5 0 0 1)
    NUMBER_OF_STEPS=${#COMMAND_IDX[@]}
    SERVER_IP_IDX=(0 0 0 1)
    EXPECTED_FAIL=("f" "f" "f" "f")
    ARG_1=("simple.txt" "simple.txt" "simple.txt" "simple.txt")
    ARG_2=("" "my_oh_god_no" "_no_no_no" "")
    for i in `seq 0 $((${NUMBER_OF_STEPS} - 1))`;
    do
        SendCommand ${COMMAND_IDX[i]} ${SERVER_IP_IDX[i]} ${EXPECTED_FAIL[i]} ${ARG_1[i]} ${ARG_2[i]}
        sleep ${WAIT_TIME}s
    done
}

RunTest3
