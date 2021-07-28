## database

- Create a MySQL database locally or with server installation
- Import table structure to MySQL database, table structure path:  vapor/toolbar/vote_reward/database/dump_reward.sql



## configuration file

- Default file name：reward.json
- A `reward.json` would look like this:

```json
{
  "node_ip": "http://127.0.0.1:9889", // node API address, replace with self node  API address
  "chain_id": "mainnet", //Node network type
  "mysql": { // Mysql connection information
    "connection": {
      "host": "192.168.30.186",
      "port": 3306,
      "username": "root",
      "password": "123456",
      "database": "reward"
    },
    "log_mode": false // default
  },
  "reward_config": {
    "xpub": "9742a39a0bcfb5b7ac8f56f1894fbb694b53ebf58f9a032c36cc22d57a06e49e94ff7199063fb7a78190624fa3530f611404b56fc9af91dcaf4639614512cb64", // Node public key (from dashboard Settings), replaced with its own
    "account_id": "bd775113-49e0-4678-94bf-2b853f1afe80", // accountID
    "password": "123456",// The password corresponding to the account ID
    "reward_ratio": 20,// The percentage of a reward given to a voter per block
    "mining_address": "sp1qfpgjve27gx0r9t7vud8vypplkzytgrvqr74rwz" // The address that receives the block reward, use the get-mining- address for mining address, for example, curl -x POST http://127.0.0.1:9889/get-mining-address -d '{}'
  }
}
```



tool use

params

```shell
distribution of reward.

Usage:
  reward [flags]

Flags:
      --config_file string         config file. default: reward.json (default "reward.json")
  -h, --help                       help for reward
      --reward_end_height uint     The end height of the distributive income reward interval, It is a multiple of the dpos consensus cycle(1200). example: 2400
      --reward_start_height uint   The starting height of the distributive income reward interval, It is a multiple of the dpos consensus cycle(1200). example: 1200
```

example:

```shell
./votereward reward --reward_start_height 6000 --reward_end_height 7200
```



Note: 

When an error (Gas credit has been spent) is returned, UTXO needs to be merged.