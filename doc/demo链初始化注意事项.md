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
    * root帐号：0x6da68a0c5dAAE0715AE6b62F00f548A2C6981c2f，beec9ec61c17b04cb9e4a9b7017e749f92835e2743e95f94cde218d667b14109
    * 目前原生币为TEST
    
* 编译
    * 安装protoc和proto-gen-go
    * 修改chain项目中的数据结构(比如transaction)，要包括core/pb/protos中的message.proto
    * proto文件修改后要用protoc来重新生成对应的go文件，具体看chain的proto_gen.sh
    * demo项目中的node.go，修改signTransaction，增加对name和type的支持
    
    
    
    
    
    
    