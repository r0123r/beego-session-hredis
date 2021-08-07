// Package hredis for session provider
//
// Usage:
// import(
//   _ "github.com/astaxie/beego/session/hredis"
//   "github.com/astaxie/beego/session"
// )
//
//	func init() {
//		globalSessions, _ = session.NewManager("hredis", ``{"cookieName":"gosessionid","gclifetime":3600,"ProviderConfig":"localhost:6379,1"}``)
//		go globalSessions.GC()
//	}
//
// more docs: http://beego.me/docs/module/session.md
package hredis

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/r0123r/beego-session-hredis/session"

	"github.com/go-redis/redis"
)

var redispder = &Provider{}
var Prefix = "session:"
var p *redis.Client

// SessionStore redis session store
type SessionStore struct {
	sid         string
	lock        sync.RWMutex
	values      map[string]interface{}
	maxlifetime int64
}

// Set value in redis session
func (rs *SessionStore) Set(key, value interface{}) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	rs.values[key.(string)] = value
	rs.save()
	return nil
}

// Get value in redis session
func (rs *SessionStore) Get(key interface{}) interface{} {
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	if v, ok := rs.values[key.(string)]; ok {
		return v
	}
	return nil
}

// Delete value in redis session
func (rs *SessionStore) Delete(key interface{}) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	delete(rs.values, key.(string))
	rs.save()
	return nil
}

// Flush clear all values in redis session
func (rs *SessionStore) Flush() error {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	rs.values = make(map[string]interface{})
	rs.save()
	return nil
}

// SessionID get redis session id
func (rs *SessionStore) SessionID() string {
	return rs.sid
}

// SessionRelease save session values to redis
func (rs *SessionStore) SessionRelease(w http.ResponseWriter) {
	rs.save()
}
func (rs *SessionStore) save() {
	//log.Printf("save:%+v\n", rs.values)
	b, err := json.Marshal(rs.values)
	if err != nil {
		log.Print(err)
		return
	}
	p.HSet(Prefix+rs.sid, "json", string(b))
	p.Expire(Prefix+rs.sid, time.Duration(rs.maxlifetime)*time.Second)
	//log.Printf("save:sid:%s:%+v\n", rs.sid, p.HGet(Prefix+rs.sid, "json").Val())
}

// Provider redis session provider
type Provider struct {
	maxlifetime int64
	savePath    string
	dbNum       int
}

// SessionInit init redis session
// savepath like redis server addr,dbnum
// e.g. 127.0.0.1:6379,0
func (rp *Provider) SessionInit(maxlifetime int64, savePath string) error {
	rp.maxlifetime = maxlifetime
	configs := strings.Split(savePath, ",")
	if len(configs) > 0 {
		rp.savePath = configs[0]
	}
	if len(configs) > 1 {
		dbnum, err := strconv.Atoi(configs[1])
		if err != nil || dbnum < 0 {
			rp.dbNum = 0
		} else {
			rp.dbNum = dbnum
		}
	} else {
		rp.dbNum = 0
	}
	p = redis.NewClient(&redis.Options{Addr: rp.savePath, DB: rp.dbNum})
	log.Print("init:", p.Time())
	return p.Ping().Err()
}

// SessionRead read redis session by sid
func (rp *Provider) SessionRead(sid string) (session.Store, error) {
	kv := make(map[string]interface{})
	jvs, err := p.HGet(Prefix+sid, "json").Bytes()
	if err == nil && jvs != nil && len(jvs) > 0 {
		var r map[string]json.RawMessage
		if err = json.Unmarshal(jvs, &r); err != nil {
			log.Print(err)
		}
		for key, val := range r {
			switch []byte(val)[0] {
			case '{', '[':
				kv[key] = val //пользовательский тип
			default:
				var i interface{}
				if err = json.Unmarshal([]byte(val), &i); err != nil {
					log.Print(err)
				}
				kv[key] = i
			}
		}
	}
	//log.Printf("read:sid:%s:%+v\n", sid, kv)
	rs := &SessionStore{sid: sid, values: kv, maxlifetime: rp.maxlifetime}
	return rs, nil
}

// SessionExist check redis session exist by sid
func (rp *Provider) SessionExist(sid string) bool {
	if existed, err := p.Exists(Prefix + sid).Result(); err != nil || existed == 0 {
		return false
	}
	return true
}

// SessionRegenerate generate new sid for redis session
func (rp *Provider) SessionRegenerate(oldsid, sid string) (session.Store, error) {
	if existed, _ := p.Exists(Prefix + oldsid).Result(); existed == 0 {
		p.HSet(Prefix+sid, "json", "")
	} else {
		p.Rename(Prefix+oldsid, Prefix+sid)
	}
	p.Expire(Prefix+sid, time.Duration(rp.maxlifetime)*time.Second)
	return rp.SessionRead(sid)
}

// SessionDestroy delete redis session by id
func (rp *Provider) SessionDestroy(sid string) error {
	return p.HDel(Prefix+sid, "json").Err()
}

// SessionGC Impelment method, no used.
func (rp *Provider) SessionGC() {
}

// SessionAll return all activeSession
func (rp *Provider) SessionAll() int {
	return 0
}

func init() {
	session.Register("hredis", redispder)
}
