package kvraft

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
	"strconv"

	"6.824/labgob"
	"6.824/labrpc"
	"6.824/raft"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}


type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	Operation string
	Key string 
	Value string
	// record this command's client and serialnumber 
	// to avoid re-executing same command that response **timeout**
	ClientId int64
	SerialNumber int
}

func (op *Op)String() string{
	var str string
	str = str + "CMD:{Operation: " + op.Operation
	str = str + ", Key: " + op.Key
	str = str + ", Value: " + op.Value
	str = str + ", ClientId: " + strconv.FormatInt(op.ClientId, 10)
	str = str + ", SerialNumber: " + strconv.Itoa(op.SerialNumber) + " }"
	return str
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	cv 		sync.Cond
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

	persister *raft.Persister   // Object to hold this peer's persisted state
	// Your definitions here.
	data map[string]string
	commands []raft.Entry 	//这里本可以只记录 command, 不记录term。但是考虑 maxraftstate, ApplyMsg需要传递 term
	snapshotIndex int
	client map[int64]int    // record latestserial number of client's command
	// nextCommandIndex int 	// point to the command to be executed 
}


func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	cmd := Op{
		Key: args.Key,
		ClientId: args.ClientId,
		Operation: args.Op,
		SerialNumber: args.SerialNumber,
	}
	_, _, isLeader := kv.rf.Start(cmd)
	if !isLeader {
		reply.Err = ErrWrongLeader
		return 
	}
	raft.Debug(raft.DServer, "S%d Get called, cmd is %v", kv.me, &cmd)
	kv.cv.L.Lock()
	// wait for apply message 
	beg := time.Now()
	for time.Since(beg).Milliseconds() < 100 { //  wait 1000 ms
		kv.cv.Wait()
		// check whether command appears in applyCh
		length := kv.snapshotIndex + len(kv.commands)
		if kv.commands[length-1].Command.(Op) != cmd {
			continue
		} 
		reply.Err = OK
		reply.Value = kv.data[cmd.Key]
		kv.cv.L.Unlock()
		return 
	}
	reply.Err = ErrTimeOut
	kv.cv.L.Unlock()
}
// 多个client调用，也都有多个 PutAppend handler运行
func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	cmd := Op{
		Key: args.Key,
		Value: args.Value,
		ClientId: args.ClientId,
		Operation: args.Op,
		SerialNumber: args.SerialNumber,
	}
	_, _, isLeader := kv.rf.Start(cmd)
	if !isLeader {
		reply.Err = ErrWrongLeader
		return 
	}
	raft.Debug(raft.DServer, "S%d PutAppend called, cmd is %v", kv.me, &cmd)
	// wait for apply message 
	kv.cv.L.Lock()
	beg := time.Now()
	for time.Since(beg).Milliseconds() < 1000 { //  wait 1000 ms
		kv.cv.Wait()
		length := kv.snapshotIndex + len(kv.commands)
		if kv.commands[length-1].Command.(Op) != cmd {
			continue
		} 
		reply.Err = OK
		kv.cv.L.Unlock()
		return 
	}
	reply.Err = ErrTimeOut
	kv.cv.L.Unlock()
}

func (kv *KVServer) Execute(cmd Op) {
	clientId := cmd.ClientId
	seqNumber := cmd.SerialNumber
	if seqNumber <= kv.client[clientId] { // 判断可能的重复命令
		return 
	}
	kv.client[clientId] = seqNumber
	key := cmd.Key
	value := cmd.Value
	raft.Debug(raft.DServer, "S%d execute %v", kv.me, cmd)
	raft.Debug(raft.DServer, "S%d client:%d seqNumber --> %v", kv.me, clientId, seqNumber)
	switch cmd.Operation{
	case GET:	// do nothing 
		return 
	case PUT:	//
		kv.data[key] = value
	case APP: 	//
		val := kv.data[key]
		kv.data[key] = val + value
	}
}

func (kv *KVServer) DecodeSnapShot(snapshot []byte) {

}
// 请问下， lab3 我思路不清，我觉得底层的raft提供一个正确的抽象，那么lab3的问题就是
// server如何处理和client，raft的交互，server接收到一个命令，调用raft.Start(), 
// server 不能保证 这个命令之后会出现在Applych里，那server什么时候回复client呢

// go routine, read committed msgs from applyCh
func (kv *KVServer) ReadApply() {
	for !kv.killed() {
		msg := <- kv.applyCh
		raft.Debug(raft.DServer, "S%d receive command %v, commandValid is %v", kv.me, msg.Command.(Op), msg.CommandValid)
		if msg.CommandValid {
			ent := raft.Entry {
				Command: msg.Command,
				Term: msg.CommandTerm,
			}
			kv.Execute(ent.Command.(Op))	// 执行命令，
			kv.mu.Lock()
			kv.commands = append(kv.commands, ent)// record commands that have been executed 
			kv.mu.Unlock()
			if _, isLeader := kv.rf.GetState(); isLeader {
				kv.cv.Broadcast() // Leader Node wake up all
			}
		} else if msg.SnapshotValid {
			kv.DecodeSnapShot(msg.Snapshot)
			kv.snapshotIndex = msg.SnapshotIndex
		}
	}
}
//
// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
//
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate
	kv.cv = sync.Cond {
		L: &kv.mu,
	}

	// You may need initialization code here.

	kv.client = make(map[int64]int)
	kv.data = make(map[string]string)

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)
	kv.persister = persister
	snapshot := kv.persister.ReadSnapshot()
	kv.DecodeSnapShot(snapshot)
	// You may need initialization code here.
	go kv.ReadApply()
	return kv
}
