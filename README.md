| LENGTH 32bit |
| STEAM ID 32 bit |
| MESSAGE FLAG 8 BIT |
| ROUTE SIZE 16bit |
| ROUTE |
| HEADER SIZE 24bit |
| HEADER DATA |
| PAYLOAD |


### request
LENGTH 字段32bits，包括数据剩余部分字节大小，包含LENGTH自身长度
STEAM ID 字段32bits，数据包的seqId
MESSAGE FLAG 字段8bits，消息类型
ROUTE SIZE 字段16bits
ROUTE 
HEADER SIZE 字段24bits，从第13个字节开始到PAYLOAD前，header字节长度
HEADER DATA KEY=VALUE&KEY=VALUE
PAYLOAD 数据

### response
LENGTH 字段32bits，包括数据剩余部分字节大小，包含LENGTH自身长度
STEAM ID 字段32bits，数据包的seqId
MESSAGE FLAG 字段8bits，消息类型
CODE 字段4字节
CODE DESC SIZE 字段2字节
CODE DESC
HEADER SIZE 字段24bits，从第13个字节开始到PAYLOAD前，header字节长度
HEADER DATA KEY=VALUE&KEY=VALUE
PAYLOAD 数据
