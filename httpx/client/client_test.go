package client

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestClient_Do(t *testing.T) {
	//cert, err := tls.LoadX509KeyPair("testdata/example-cert.pem", "testdata/example-key.pem")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//tlscfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	callInfo := &HeaderCall{}
	callInfo.Add("Content-Type", "application/json")
	client, err := NewClient(
		WithEndpoint("127.0.0.1:8081"),
		WithTimeOut(1*time.Second),
		//WithTlsConf(tlscfg),
	)
	assert.Nil(t, err)
	if err != nil {
		t.Log(err)
	}
	body, err := client.Do(context.Background(), "GET", "/test", []byte{}, callInfo)
	assert.Nil(t, err)
	if err != nil {
		t.Log(err)
	}
	fmt.Println(string(body))
}

// 测试并发效率
func BenchmarkClientParallel(b *testing.B) {
	callInfo := &HeaderCall{}
	callInfo.Add("Content-Type", "application/json")
	client, err := NewClient(
		WithEndpoint("127.0.0.1:8081"),
		WithTimeOut(1*time.Second),
	)
	assert.Nil(b, err)
	if err != nil {
		b.Log(err)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			body, err := client.Do(context.Background(), "GET", "/test", []byte{}, callInfo)
			assert.Nil(b, err)
			if err != nil {
				b.Log(err)
			}
			fmt.Println(string(body))
		}
	})
}
