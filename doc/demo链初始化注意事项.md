## 配置文件
* 格式
```
{
    "address":"localhost",
    "port":7001,
    "rpc_address":"localhost",
    "rpc_port":7545,
    "gas_address":"0xc287B1266732495Fe8c93CE3Ba631597153fdd91",
    "data_dir":"./data",
    "secret":"abb33a8c2bb48d3b1c2ce365685ac3b96563e6a17ebc4367cd637e33149f94ea",
    "bootstraps":[
        "localhost:7001",
        "localhost:7002",
        "localhost:7003",
        "localhost:7004",
        "localhost:7005"
    ]
}
```
* 注意
    * 每个节点的gas_address必须一致
    * 每个节点的secret必须不一致
    * 目前账本的地址格式是使用以太坊的地址格式（比如0xc287B1266732495Fe8c93CE3Ba631597153fdd91）， 而非井通的格式（比如jHb9CJAWyB4jr91VRWn96DkukG4bwdtyTh），可以通过插拔账本模块来改变