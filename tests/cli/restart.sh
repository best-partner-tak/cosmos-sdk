#!/bin/bash

# these are two globals to control all scripts (can use eg. counter instead)
SERVER_EXE=basecoin
CLIENT_EXE=basecli

oneTimeSetUp() {
  # these are passed in as args
  BASE_DIR=$HOME/.basecoin_test_restart
  CHAIN_ID=restart-chain

  rm -rf $BASE_DIR 2>/dev/null
  mkdir -p $BASE_DIR

  # set up client - make sure you use the proper prefix if you set
  # a custom CLIENT_EXE
  export BC_HOME=${BASE_DIR}/client
  prepareClient

  # start basecoin server (with counter)
  initServer $BASE_DIR $CHAIN_ID 3456
  if [ $? != 0 ]; then return 1; fi

  initClient $CHAIN_ID 34567

  echo "...Testing may begin!"
  echo
  echo
  echo
}

oneTimeTearDown() {
  echo
  echo
  stopServer $PID_SERVER
}

test00PreRestart() {
  SENDER=$(getAddr $RICH)
  RECV=$(getAddr $POOR)

  RES=$(echo qwertyuiop | ${CLIENT_EXE} tx send --amount=992mycoin --sequence=1 --to=$RECV --name=$RICH 2>/dev/null)
  txSucceeded $? "$RES"
  TX=`echo $RES | cut -d: -f2-`
  HASH=$(echo $TX | jq .hash | tr -d \")
  TX_HEIGHT=$(echo $TX | jq .height)

  checkAccount $SENDER "1" "9007199254740000"
  checkAccount $RECV "0" "992"

  # make sure tx is indexed
  checkSendTx $HASH $TX_HEIGHT $SENDER "992"

}


test01OnRestart() {
  SENDER=$(getAddr $RICH)
  RECV=$(getAddr $POOR)

  RES=$(echo qwertyuiop | ${CLIENT_EXE} tx send --amount=10000mycoin --sequence=2 --to=$RECV --name=$RICH 2>/dev/null)
  txSucceeded $? "$RES"
  if [ $? != 0 ]; then echo "can't make tx!"; return 1; fi

  TX=`echo $RES | cut -d: -f2-`
  HASH=$(echo $TX | jq .hash | tr -d \")
  TX_HEIGHT=$(echo $TX | jq .height)

  # wait til we have quite a few blocks... like at least 20,
  # so the query command won't just wait for the next eg. 7 blocks to verify the result
  echo "waiting to generate lots of blocks..."
  sleep 20
  echo "done waiting!"

  # last minute tx just at the block cut-off...
  RES=$(echo qwertyuiop | ${CLIENT_EXE} tx send --amount=20000mycoin --sequence=3 --to=$RECV --name=$RICH 2>/dev/null)
  txSucceeded $? "$RES"
  if [ $? != 0 ]; then echo "can't make second tx!"; return 1; fi

  # now we do a restart...
  stopServer $PID_SERVER
  startServer $BASE_DIR/server $BASE_DIR/${SERVER_EXE}.log
  if [ $? != 0 ]; then echo "can't restart server!"; return 1; fi

  # make sure queries still work properly, with all 3 tx now executed
  checkAccount $SENDER "3" "9007199254710000"
  checkAccount $RECV "0" "30992"

  # make sure tx is indexed
  checkSendTx $HASH $TX_HEIGHT $SENDER "10000"

  # for double-check of logs
  if [ -n "$DEBUG" ]; then
    cat $BASE_DIR/${SERVER_EXE}.log;
  fi
}


# load and run these tests with shunit2!
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )" #get this files directory

# load common helpers
. $DIR/common.sh

. $DIR/shunit2

