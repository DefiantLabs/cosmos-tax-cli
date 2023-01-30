# dev info

## Bootstrap Script

clone https://github.com/CosmosContracts/juno.git and make sure `docker-compose up` works.  Stop with `docker-compose down`. Make sure you have the required binaries like `jq` installed and on your path. Download this file to the juno repo you just cloned, and run this shell script.


```
#!/bin/bash

## Generate a test-user key

echo "Y" | junod keys delete test_user_wallet 2> /dev/null
export TEST_USER_PASS="testuserpass"
echo "$TEST_USER_PASS" | junod keys add test_user_wallet
echo "$TEST_USER_PASS" | junod keys export test_user_wallet 2> test_user_wallet.key


## Generate a sample data

cat <<EOF > sample_data.sh
#!/bin/sh
sleep 10

# Import Key

echo \$TEST_USER_WALLET_PASS | junod keys import \$TEST_USER_WALLET_NAME \$TEST_USER_WALLET_KEY
export VALOPER_ADDRESS=\$(junod keys show validator -a --bech val)
export TEST_USER_ADDRESS=\$(junod keys show \$TEST_USER_WALLET_NAME -a)

# Put Test Transactions Here
# Note that the sleep command is needed sometimes after transactions
echo "Y" | junod tx bank send validator \$TEST_USER_ADDRESS 80085ustake --chain-id testing
sleep 2
echo "Y" | junod tx staking delegate \$VALOPER_ADDRESS 50000ustake --from \$TEST_USER_ADDRESS --chain-id testing
sleep 2
echo "Y" | junod tx staking delegate \$VALOPER_ADDRESS 85ustake --from \$TEST_USER_ADDRESS --chain-id testing
sleep 2
junod query bank balances \$TEST_USER_ADDRESS

EOF

## Create test-case compose file.

cat <<EOF > test-case.yml
version: "3"
services:
  node:
    environment:
    - TEST_DATA_FILE=\${TEST_DATA_FILE}
    - TEST_USER_WALLET_NAME=\${TEST_USER_WALLET_NAME}
    - TEST_USER_WALLET_KEY=\${TEST_USER_WALLET_KEY}
    - TEST_USER_WALLET_PASS=\${TEST_USER_WALLET_PASS}
    volumes:
      - ./\${TEST_USER_WALLET_KEY}:/opt/\${TEST_USER_WALLET_KEY}
      - ./\${TEST_DATA_FILE}:/opt/\${TEST_DATA_FILE}
    entrypoint: ["/bin/sh", "-c"]
    command:
      - |
        chmod 755 /opt/\${TEST_DATA_FILE}
        /opt/\${TEST_DATA_FILE} &
        ./setup_and_run.sh juno16g2rahf5846rxzp3fwlswy08fz8ccuwk03k57y

EOF


## Create compose wrapper for custom test cases.


cat <<EOF > sample.env
COMPOSE_FILE=docker-compose.yml:test-case.yml
TEST_DATA_FILE=sample_data.sh
TEST_USER_WALLET_NAME=test_user_wallet
TEST_USER_WALLET_KEY=test_user_wallet.key
TEST_USER_WALLET_PASS=testuserpass
EOF


echo "Start sample data juno node now with:"
echo "docker-compose --env-file sample.env up -d"
```
