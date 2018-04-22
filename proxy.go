package main

import (
    "os"
    "github.com/coreos/etcd/client"
    "time"
    "log"
    "encoding/json"
    "context"
)

type Service struct {
    ProcessId int            // 进程ID ， 单机调试时用来标志每一个服务
    Info      ServerInf     // 服务端信息
    KeysAPI   client.KeysAPI // API client, 此处用的是 V2 版本的API，是基于     http 的。 V3版本的是基于grpc的API
}

type ServerInf struct {
    Id   int32  `json:"id"`   // 服务器ID
    IP   string `json:"ip"`   // 对外连接服务的 IP
    Port int32  `json:"port"` // 对外服务端口，本机或者端口映射后得到的
}

// 注册服务
func RegisterService(endpoints []string) {
    cfg := client.Config{
        Endpoints:               endpoints,
        Transport:               client.DefaultTransport,
        HeaderTimeoutPerRequest: time.Second,
    }

    etcd, err := client.New(cfg)
    if err != nil {
        log.Fatal("Error: cannot connec to etcd:", err)
    }

    s := &Service{
        ProcessId: os.Getpid(),
        Info:      ServerInf{Id: 1024, IP: "123.59.204.170", Port: 9009},
        KeysAPI:   client.NewKeysAPI(etcd),
    }
    s.HeartBeat()
}

func (s *Service) HeartBeat() {
    api := s.KeysAPI
    for {
        key := "/v1/login"
        value, _ := json.Marshal(s.Info)

        _, err := api.Set(context.Background(), key, string(value), &client.SetOptions{
            TTL: time.Second * 20,
        })

        if err != nil {
            log.Println("Error update workerInfo:", err)
        }
        time.Sleep(time.Second * 10)
    }
}

func main() {
    endPoints := []string{"http://192.168.99.100:2379", "http://192.168.99.101:2379", "http://192.168.99.102:2379"}
    RegisterService(endPoints)
}
