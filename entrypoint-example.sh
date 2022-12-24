#!/bin/sh
sleep 5
cosmos-tax-cli-private index \
      --log.pretty = true \
      --log.level = info \
      --base.startBlock 13148974 \
      --base.endBlock -1 \
      --lens.accountPrefix cosmos \
      --lens.keyringBackend test \
      --lens.chainID cosmoshub-4 \
      --lens.chainName cosmoshub \
      --lens.rpc https://rpc-cosmoshub.blockapsis.com:443 \
      --db.host postgres \
      --db.database postgres \
      --db.user taxuser \
      --db.password password