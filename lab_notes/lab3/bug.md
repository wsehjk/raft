## basic3a timed out
basic3a 发现超时错误，日志显示Leader 调用`start`()之后，rpc handler还没有`wait()`，命令就出现在applyCh中，broadcast没有生效，之后`wait()`也没办法被唤醒了， wait()不能被唤醒，server也没办法告诉client超时了，结果就是超时错误，那么必须让 1）`wait`在`broadcast`之前调用，这样`broadcast`才能有效，2）让`wait`等待一段时间自行退出。`go`目前还不支持`wait on timed out`。使用方法1 就需要在调用`start`和`broadcast`之前申请条件变量锁，这样能保证`broadcast`在`wait`之后执行。流程是这样的 `broadcast`能执行说明获取了条件变量的锁，说明`wait`释放了条件变量的锁，这样就说明先`wait`后有`broadcast`
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
## TestSpeed3A 性能问题
原本实现中日志只会在`toker`中以一个心跳的频率发给`Follower`，这样效率不高。新的命令在`start`调用之后就会出现在`log`中，这是就可以给`Follower`发送日志信息。所以在`start()`中加`goroutine`及时发送，提交效率。
之后重新测试`2d`和`2c`还是发现问题
1. 2d: index out of range[-1]
   问题出现在 `server`调用`Snapshot`，结果`index = rf.lastIncludedIndex`, `index`怎么会和`rf.lastIndecludeIndex`相同，感觉是直接修改了`rf.lastIncludedIndex`，那应该是`InstallSnapshot` hanler了。

   ```go
    func (rf *Raft) Snapshot(index int, snapshot []byte) {
        rf.mu.Lock()
        defer rf.mu.Unlock()
        Debug(dSnap, "S%d snapshot called, index: %d, rf.lastIncludedIndex %d", rf.me, index, rf.lastIncludedIndex)
        rf.lastIncludedTerm = rf.logs[index-rf.lastIncludedIndex - 1].Term // index out of range
        ....
    }
   ```
2. 2d: server apply out of order, expected index 50, got 48
    `Leader`没有收到`Follower`的回复信息，认为`Follower lags behind`, `Leader`向`Follower`发送`snapshot`。此时`Follower` 正在向`server`提交信息，`Leader`发送的`snapshot`会打乱这个顺序报错。解决方法是 `raft`记录`applying`这个原子变量，表示节点正在提交，此时不应该接受`snapshot`。

    `applying`为`false`时，表示稳定状态，可以接受`snapshot`。此时`snapshot.LastIncludedIndex`应该`> rf.commitIndex`。日志分析表明`server`不会随机调用`snapshot()`，每10个`log`调用一次`snapshot index%10 = 9`，说明每个`raft`节点`snapshot index`都相同，且每个`commitindex` 都有唯一的`lastincludededindex`。如果`snapshot.LastIncludedIndex <= rf.commitIndex`，说明这个`LastIncludedIndex`要么是之前出现过，即过时的`index`，要么是`rf.commitIndex`对应的`lastincludeindex`，如果接受`snapshot`，之后`server`调用`snapshot(index) `就会`index = rf.LastIncludedIndex`，出现`out of range error`

## TestConcurrent3A timed out 
这个测试在可靠的环境下进行，没有延迟，没有crash，但是超时日志显示发生了重新选举。