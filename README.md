# beego-session-hredis
Package hredis(used hset,json_encode) for beego session provider

 ## Usage:
 	import(
   	_ "github.com/astaxie/beego/session/hredis"
   	"github.com/astaxie/beego/session"
 	)

	func init() {
		globalSessions, _ = session.NewManager("hredis", ``{"cookieName":"gosessionid","gclifetime":3600,"ProviderConfig":"localhost:6379,1"}``)
		go globalSessions.GC()
	}

 more docs: http://beego.me/docs/module/session.md
