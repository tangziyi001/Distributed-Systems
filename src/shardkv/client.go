package shardkv

//
// client code to talk to a sharded key/value service.
//
// the client first talks to the shardmaster to find out
// the assignment of shards (keys) to groups, and then
// talks to the group that holds the key's shard.
//

import "labrpc"
import "crypto/rand"
import "math/big"
import "shardmaster"
import "time"
import "sync"
import "fmt"

//
// which shard is a key in?
// please use this function,
// and please do not change it.
//
func key2shard(key string) int {
	shard := 0
	if len(key) > 0 {
		shard = int(key[0])
	}
	shard %= shardmaster.NShards
	return shard
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

type Clerk struct {
	sm       *shardmaster.Clerk
	config   shardmaster.Config
	make_end func(string) *labrpc.ClientEnd
	// You will have to modify this struct.
	debug bool
	id int64
	count int
	mu sync.Mutex
}

//
// the tester calls MakeClerk.
//
// masters[] is needed to call shardmaster.MakeClerk().
//
// make_end(servername) turns a server name from a
// Config.Groups[gid][i] into a labrpc.ClientEnd on which you can
// send RPCs.
//
func MakeClerk(masters []*labrpc.ClientEnd, make_end func(string) *labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.sm = shardmaster.MakeClerk(masters)
	ck.make_end = make_end
	// You'll have to add code here.
	ck.debug = false
	ck.count = 0
	ck.id = nrand()
	ck.count = 0
	return ck
}

//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
// You will have to modify this function.
//
func (ck *Clerk) Get(key string) string {
	args := GetArgs{}
	args.Key = key
	ck.mu.Lock()
	id := ck.count
	ck.count += 1
	args.ClientId = ck.id
	args.Id = id
	ck.mu.Unlock()
	if ck.debug {
		fmt.Printf("\nClient Get, key %v, id %v\n", key, id)
	}
	for {
		shard := key2shard(key)
		gid := ck.config.Shards[shard]
		if servers, ok := ck.config.Groups[gid]; ok {
			// try each server for the shard.
			for si := 0; si < len(servers); si++ {
				srv := ck.make_end(servers[si])
				var reply GetReply
				ok := srv.Call("ShardKV.Get", &args, &reply)
				if ok && reply.WrongLeader == false && (reply.Err == OK || reply.Err == ErrNoKey) {
					if ck.debug {
						fmt.Printf("\nClient Get Completed, key %v, value %v\n", key, reply.Value)
					}
					return reply.Value
				}
				if ok && (reply.Err == ErrWrongGroup) {
					break
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
		// ask master for the latest configuration.
		ck.config = ck.sm.Query(-1)
	}

	return ""
}

//
// shared by Put and Append.
// You will have to modify this function.
//
func (ck *Clerk) PutAppend(key string, value string, op string) {
	args := PutAppendArgs{}
	args.Key = key
	args.Value = value
	args.Op = op
	ck.mu.Lock()
	id := ck.count
	ck.count += 1
	args.ClientId = ck.id
	args.Id = id
	ck.mu.Unlock()
	retry := 0
	if ck.debug {
		fmt.Printf("\nClient PutAppend, key %v, value %v, id %v\n", key, value, id)
	}
	for {
		if ck.debug {
			fmt.Printf("\nClient PutAppend, key %v, value %v, id %v, retry %d\n", key, value, id, retry)
			retry += 1
		}
		shard := key2shard(key)
		gid := ck.config.Shards[shard]
		if servers, ok := ck.config.Groups[gid]; ok {
			for si := 0; si < len(servers); si++ {
				srv := ck.make_end(servers[si])
				var reply PutAppendReply
				ok := srv.Call("ShardKV.PutAppend", &args, &reply)
				if ok && reply.WrongLeader == false && (reply.Err == OK || reply.Err == ErrOutdated) {
					if ck.debug {
						fmt.Printf("\nClient PutAppend Completed, key %v, value %v, id %v\n", key, value, id)
					}
					return
				}
				if ok && reply.Err == ErrWrongGroup {
					if ck.debug {
						fmt.Printf("\nClient PutAppend Wrong Group, key %v, value %v, id %v\n", key, value, id)
					}
					break
				} else if ok {
					if ck.debug {
						fmt.Printf("\nClient PutAppend OK, key %v, value %v, id %v, WrongLeader %v, Err %v\n", key, value, id, reply.WrongLeader, reply.Err)
					}
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
		// ask master for the latest configuration.
		ck.config = ck.sm.Query(-1)
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
