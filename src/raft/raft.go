package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
	"6.824/labrpc"
)

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

const (
	Leader     = "Leader"
	Follower   = "Follower"
	Candidate  = "Candidate"
	HeartbeatInterval = 100  //ms 
) 

type entry struct {
	Term int
	Command interface{}
} 
//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	votedFor int
	currentTerm int
	logs []entry

	role string
	timer int

	lastApplied int
	commitIndex int
	nextIndex []int
	matchIndex []int
	applyCh chan ApplyMsg
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	term = rf.currentTerm
	isleader = rf.role == Leader
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}


//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}


//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}


//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Candidate int
	Term int
	LastLogTerm int
	LastLogIndex int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Granted bool
	Term int
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Granted = false
		return 
	}
	if args.Term == rf.currentTerm && rf.votedFor != -1 {
		reply.Term = rf.currentTerm
		reply.Granted = false
		return	
	}

	// check election restriction 
	lastLogIndex := len(rf.logs)
	lastLogTerm := 0
	if lastLogIndex != 0 {
		lastLogTerm = rf.logs[lastLogIndex-1].Term
	}
	if args.LastLogTerm < lastLogTerm {
		reply.Term = rf.currentTerm
		reply.Granted = false
		return	
	}
	if args.LastLogTerm == lastLogTerm && args.LastLogIndex < lastLogIndex {
		reply.Term = rf.currentTerm
		reply.Granted = false
		return		
	}

	rf.currentTerm = args.Term
	rf.votedFor = args.Candidate
	rf.persist()
	rf.role = Follower
	rf.timer = GetTimer()  // reset timer because this server votes for candidate 

	reply.Term = rf.currentTerm
	reply.Granted = true
	Debug(dVote, "S%d votes for S%d", rf.me, args.Candidate)
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

//
// append entry rpc arguments
//
type AppendEntryArgs struct {
	Term int
	LeaderId int
	Logs []entry
	PrevLogIndex int
	PrevLogTerm  int
	LeaderCommitIndex int
}

//
// append entry rpc replies
//
type AppendEntryReply struct {
	Accept bool
	Term int

	Xterm int
	Xindex int
	Len int
}

// 
// Append entry rpc handler 
// 
func (rf *Raft) AppendEntry(args *AppendEntryArgs, reply *AppendEntryReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Accept = false
		return
	}
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.votedFor = -1
		rf.persist()
		rf.role = Follower
	}
	rf.timer = GetTimer()   // recevie append entries from valid Leader 
	if len(args.Logs) == 0 {  // heartbeat with empty entries
		reply.Term = rf.currentTerm
		reply.Accept = true
		return
	}

	if  args.PrevLogIndex > len(rf.logs) || 
		rf.logs[args.PrevLogIndex - 1].Term != args.PrevLogTerm {
		reply.Term = rf.currentTerm
		reply.Accept = false
		return
	}

	reply.Term = rf.currentTerm
	reply.Accept = true

	i := args.PrevLogIndex - 1
	j := 0
	for i < len(rf.logs) && j < len(args.Logs)  {
		if rf.logs[i] != args.Logs[j] {
			break
		}
		i+=1 
		j+=1
	}
	rf.logs = rf.logs[0:i]
	rf.logs = append(rf.logs, args.Logs[j:]...)
	rf.commitIndex = len(rf.logs)
	if args.LeaderCommitIndex < rf.commitIndex {
		rf.commitIndex = args.LeaderCommitIndex
	}
	for rf.lastApplied < rf.commitIndex {
		go func(index int) {
			msg := ApplyMsg{
				CommandValid: true,
				Command: rf.logs[index].Command,
				CommandIndex: index + 1,
			}
			rf.applyCh <- msg
		}(rf.lastApplied)
		rf.lastApplied ++
	}
}
//
// send append entry rpc 
// 
func (rf *Raft) sendAppendEntry(server int, args *AppendEntryArgs, reply *AppendEntryReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntry", args, reply)
	return ok
}
//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).


	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for !rf.killed() {
		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		time.Sleep(10 * time.Millisecond)
		rf.mu.Lock()
		if rf.role == Leader {
			rf.mu.Unlock()
			continue 
		}
		rf.timer -= 10
		if (rf.timer > 0 && rf.role == Follower) {
			rf.mu.Unlock()
			continue 	
		}

		// initiate another round of election 
		if rf.timer <= 0 { 
			rf.currentTerm += 1
			rf.votedFor = rf.me   // vote for it self
			rf.persist()
			rf.timer = GetTimer()
		}
		
		rf.role = Candidate
		votes := 1    // number of votes get 

		candidate := rf.me
		term := rf.currentTerm
		lastLogIndex := len(rf.logs)	
		lastLogTerm := 0
		// if rf.logs is empty, then lastLogIndex is 0
		// lastLogTerm is 0, denoting the least up-to-date 
		if lastLogIndex != 0 { 
			lastLogTerm = rf.logs[lastLogIndex-1].Term
		}

		tot := len(rf.peers)
		rf.mu.Unlock()
		for i:= range rf.peers {
			if i == candidate {
				continue
			}

			go func(peer int){ // send rpc parallel
				args := RequestVoteArgs{
					Term : term,
					Candidate : candidate,
					LastLogTerm : lastLogTerm,
					LastLogIndex : lastLogIndex,
				}
				reply := RequestVoteReply{}
	
				ok := rf.sendRequestVote(peer, &args, &reply)

				if !ok {
					Debug(dError, "requestvote rpc reply error")
					return 
				}

				rf.mu.Lock()
				defer rf.mu.Unlock()
				if rf.role != Candidate {
					return 
				}
				if reply.Granted {
					votes += 1
					if votes >= (tot+1)/2 {
						rf.role = Leader
						rf.nextIndex = make([]int, len(rf.peers))
						for i := range rf.nextIndex {
							rf.nextIndex[i] = len(rf.logs) + 1
						}
						rf.matchIndex = make([]int, len(rf.peers))
						Debug(dRole, "S%d becomes Leader, gets %d votes", candidate, votes)

						// send HeartBeats immediately 
						for i := range rf.peers {
							if i == candidate {
								continue
							}
							AppendArgs := AppendEntryArgs{
								Term: term,	
								LeaderId: candidate,
							}
							AppendReply := AppendEntryReply{}
							go rf.sendAppendEntry(i, &AppendArgs, &AppendReply)
						}
					} 
				} else if reply.Term > term {
					rf.role = Follower
					rf.currentTerm = reply.Term
					rf.votedFor = -1
					rf.persist()
					Debug(dRole, "S%d becomes Follower, term: %d --> %d", candidate, term, reply.Term)
				} 
			}(i)
		}
	}
}

// The applier go routine sends append entries and heartbeats to follower 
func (rf *Raft) applier() {
	for !rf.killed() {
		time.Sleep(HeartbeatInterval * time.Millisecond)  // heartbeat interval 
		rf.mu.Lock()
		if rf.role != Leader {
			rf.mu.Unlock()
			continue
		}
		// only leader allowed to send appen entries, though maybe outdated 
		// send append entries or heartbeats 
		logs := make([]entry, len(rf.logs))   // read only
		copy(logs, rf.logs)
		nextIndex := make([]int, len(rf.nextIndex))
		copy(nextIndex, rf.nextIndex)		  // read only 
		commitIndex := rf.commitIndex
		term := rf.currentTerm
		leaderId := rf.me
		rf.mu.Unlock()

		for i := range rf.peers {
			if i == leaderId {
				continue
			}
			go func(server int) {
				reply := AppendEntryReply{}
				args := AppendEntryArgs{
					Term : term,
					LeaderId : leaderId,
					LeaderCommitIndex : commitIndex,
				}

				// has more comands to replica
				if len(logs) >= nextIndex[server] {
					args.PrevLogIndex = nextIndex[server] - 1
					args.PrevLogTerm = 0
					if args.PrevLogIndex != 0 {
						args.PrevLogTerm = logs[args.PrevLogIndex-1].Term
					}
					args.Logs = logs[args.PrevLogIndex:]
				}

				ok := rf.sendAppendEntry(server, &args, &reply)
				if !ok {
					Debug(dError, "appendEntry rpc reply error")
					return 
				}

				rf.mu.Lock()  //relock
				defer rf.mu.Unlock()
				if rf.role != Leader {
					return 
				}
				if reply.Accept {  // accept 
					rf.nextIndex[server] = len(logs) + 1
					rf.matchIndex[server] = len(logs)
					Debug(dLeader, "S%d [Leader] nextindex[%d]: %d, matchIndex[%d]: %d", rf.me, server, len(logs) + 1, server, len(logs))
				} else if reply.Term > rf.currentTerm {  // outdate leader 
					rf.currentTerm = reply.Term
					rf.votedFor = -1
					rf.persist()
					rf.role = Follower
					Debug(dRole, "S%d [Leader] becomes Follower receiving append entry reply", rf.me)
				} else {
					nextIndex[server] -= 1    // send one append entry rpc per entry
				}
			}(i)
		}
	}
}
//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.currentTerm = 0
	rf.role = Follower
	rf.votedFor = -1
	rf.timer = GetTimer()

	rf.lastApplied = 0
	rf.commitIndex = 0
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	rf.applyCh = applyCh
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	for i := range rf.nextIndex {
		rf.nextIndex[i] = len(rf.logs) + 1
	}
	Debug(dTest, "S%d make ", me)		
	// start ticker goroutine to start elections
	go rf.ticker()

	// start applier goroutine to send append entries
	go rf.applier()

	return rf
}
