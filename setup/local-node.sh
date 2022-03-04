#!/bin/sh

# NOTE: this is a work in progress and for now only shows working commands, but does not setup a working node.

# first we need to set up the keys.
# this is the mnemonic of a test key 'kyle' that will be used for testing transactions that send/receive funds.
# When prompted you will need to paste it in
# cup lend senior velvet sleep rely stock roast area color violin such urban endless strategy such more future crane cruel tone daring fly style
junod keys add kyle --recover

# this is the mnemonic of the Juno development team's test user key. it is a genesis account.
# When prompted you will need to paste it in
# clip hire initial neck maid actor venue client foam budget lock catalog sweet steak waste crater broccoli pipe steak sister coyote moment obvious choose
junod keys add default --recover

# Launch the node in the background
docker-compose up -d
# Give the node time to startup in case this is first run
sleep 10 

# send some money from the genesis key to our new key (juno1m2hg5t7n8f6kzh8kmh98phenk8a4xp5wyuz34y=the kyle key from above)
junod tx bank send default juno1m2hg5t7n8f6kzh8kmh98phenk8a4xp5wyuz34y 1000ustake --chain-id testing
# show balances
junod query bank balances juno1m2hg5t7n8f6kzh8kmh98phenk8a4xp5wyuz34y --chain-id testing

#Validator address, based on the defaults used by the Juno development team:
#juno197nlqz8q6jfm83s3fmmznd6pnvxuxp6zygv26p
#junovaloper197nlqz8q6jfm83s3fmmznd6pnvxuxp6zm469pc
junod tx staking delegate junovaloper197nlqz8q6jfm83s3fmmznd6pnvxuxp6zm469pc 50000ustake --from kyle --chain-id testing