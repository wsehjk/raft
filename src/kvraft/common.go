package kvraft

import "fmt"

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongLeader = "ErrWrongLeader"
	ErrTimeOut     = "ErrTimeOut"
)

const (
	GET = "Get"
	PUT = "Put"
	APP = "Append"
)

type Err string

// Put or Append
type PutAppendArgs struct {
	Key   string
	Value string
	Op    string // "Put" or "Append"
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	ClientId int64
	SerialNumber int
}

func (args *PutAppendArgs)String() string {
	return fmt.Sprintf("PutAppendArgs {ClientId: %d, SerialNumber: %d}", args.ClientId, args.SerialNumber) 
}


type PutAppendReply struct {
	Err Err
}

type GetArgs struct {
	Key string
	// You'll have to add definitions here.
	Op string  // Get
	ClientId int64
	SerialNumber int
}

func (args *GetArgs)String() string {
	return fmt.Sprintf("GetArgs {ClientId: %d, SerialNumber: %d}", args.ClientId, args.SerialNumber) 
}
type GetReply struct {
	Err   Err
	Value string
}
