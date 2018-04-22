package main

import (
    "log"
    "net/http"
    "net/http/httputil"
    "github.com/coreos/etcd/client"
    "net/url"
    "sync"
    "os"
    "context"
    "encoding/json"
    "fmt"
)

type ServerInfo struct {
    Id   int32  `json:"id"`   // 服务器ID
    IP   string `json:"ip"`   // 对外连接服务的 IP
    Port int32  `json:"port"` // 对外服务端口，本机或者端口映射后得到的
}

type Server struct {
    Info   ServerInfo // 服务器信息
    Status int        // 服务器状态
}

// all server
type ServerPool struct {
    services map[string]*Server // server 列表
    client   client.Client
    mu       sync.RWMutex
}

var (
    DefaultPool ServerPool
    once        sync.Once
)

func Init(endpoints []string) {
    once.Do(func() { DefaultPool.init(endpoints) })
}

func (p *ServerPool) init(hosts []string) {
    cfg := client.Config{
        Endpoints: hosts, //
        Transport: client.DefaultTransport,
    }
    cli, err := client.New(cfg)
    if err != nil {
        log.Panic(err)
        os.Exit(-1)
    }
    p.client = cli

    p.services = make(map[string]*Server)

    go p.watcher()
}

func (p *ServerPool) addServer(key, value string) {
    log.Printf(value)
    var info ServerInfo
    if err := json.Unmarshal([]byte(value), &info); err != nil {
        log.Printf("%v", err)
        return
    }

    p.mu.RLock()
    defer p.mu.RUnlock()
    ser := Server{
        Status: 200,
        Info:   info,
    }

    p.services[key] = &ser

}

func (p *ServerPool) removeServer(key string) {

    if _, ok := p.services[key]; ok {
        p.mu.Lock()
        defer p.mu.Unlock()
        delete(p.services, key)
    }

}

func (p *ServerPool) watcher() error {
    kAPI := client.NewKeysAPI(p.client)
    w := kAPI.Watcher("/v1/login", &client.WatcherOptions{Recursive: true})
    for {
        resp, err := w.Next(context.Background())
        if err != nil {
            log.Println(err)
            continue
        }
        if resp.Node.Dir {
            continue
        }
        switch resp.Action {
        case "set", "create", "update", "compareAndSwap":
            log.Printf("here is update")
            p.addServer(resp.Node.Key, resp.Node.Value)
        case "delete", "compareAndDelete", "expire":
            p.removeServer(resp.PrevNode.Key)
        }
    }
}

func main() {
    endPoints := []string{"http://192.168.99.100:2379", "http://192.168.99.101:2379", "http://192.168.99.102:2379"}
    Init(endPoints)
    server := http.Server{
        Addr: ":8080",
        Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            serName := r.URL.Path
            if ser, ok := DefaultPool.services[serName]; ok {
                tag := fmt.Sprintf("http://%s:%d/", ser.Info.IP, ser.Info.Port)
                target, err := url.Parse(tag)
                if err != nil {
                    log.Fatalf("could not parse URL: %s", err)
                }
                proxy := httputil.NewSingleHostReverseProxy(target)
                proxy.ServeHTTP(w, r)

            } else {
                w.WriteHeader(http.StatusNotFound)
            }

        }),
    }
    err := server.ListenAndServe()
    if err != nil {
        log.Fatalf("could not serve: %s", err)
    }

}
