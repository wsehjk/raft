## basic3a timed out
basic3a 发现超时错误，日志显示Leader 调用`start`()之后，rpc handler还没有wait()，命令就出现在applyCh中，broadcast没有生效，之后wait()也没办法被唤醒了， wait()不能被唤醒，server也没办法告诉client超时了，结果就是超时错误，那么必须让 1）`wait`在`broadcast`之前调用，这样`broadcast`才能有效，2）让`wait`等待一段时间自行退出。`go`目前还不支持`wait on timed out`。使用方法1 就需要在调用`start`和`broadcast`之前申请条件变量锁，这样能保证`broadcast`在`wait`之后执行。流程是这样的 `broadcast`能执行说明获取了条件变量的锁，说明`wait`释放了条件变量的锁，这样就说明先`wait`后有`broadcast`
Log
``` bash
145805 LOG1 S4 [Leader: 1] length of log is 131
145870 LEAD S4 [Leader] nextindex[3]: 132
145870 LEAD S4 [Leader] matchIndex[3]: 131
145874 LEAD S4 [Leader] nextindex[0]: 132
145875 LEAD S4 [Leader] matchIndex[0]: 131
145875 CMIT S4 updateCommitIndex rf.commitIndex is 130
145875 CMIT S4 updatecommitIndex i: 130, lastIncludedIndex: 0
145876 CMIT S4 commitIndex: 131 after update
145876 INFO S4 in apply(), lastapplied 130 commitIndex 131
145876 CMIT S4 apply command {Append 0 x 0 16 y 721163820834064835 34} at index 131 end is 131
145877 SERV S4 receive command {Append 0 x 0 16 y 721163820834064835 34}, commandValid is true
145877 SERV S4 execute {Append 0 x 0 16 y 721163820834064835 34}
145878 SERV S4 client:721163820834064835 seqNumber --> 34
145820 SERV S4 PutAppend called, cmd is CMD:{Operation: Append, Key: 0, Value: x 0 16 y, ClientId: 721163820834064835, SerialNumber: 34 } // 还没有调用wait
```
## TestConcurrent3A timed out 
这个测试在可靠的环境下进行，没有延迟，没有crash，但是超时日志显示发生了重新选举。