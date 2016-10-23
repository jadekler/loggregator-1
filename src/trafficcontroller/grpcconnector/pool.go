package grpcconnector

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"plumbing"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
)

type Pool struct {
	size int

	mu       sync.RWMutex
	dopplers map[string][]unsafe.Pointer
}

type clientInfo struct {
	client plumbing.DopplerClient
	closer io.Closer
}

func NewPool(size int) *Pool {
	return &Pool{
		size:     size,
		dopplers: make(map[string][]unsafe.Pointer),
	}
}

func (p *Pool) RegisterDoppler(addr string) {
	clients := make([]unsafe.Pointer, p.size)

	p.mu.Lock()
	p.dopplers[addr] = clients
	p.mu.Unlock()

	for i := 0; i < p.size; i++ {
		go p.connectToDoppler(addr, clients, i)
	}
}

func (p *Pool) Subscribe(dopplerAddr string, ctx context.Context, req *plumbing.SubscriptionRequest) (plumbing.Doppler_SubscribeClient, error) {
	p.mu.RLock()
	clients := p.dopplers[dopplerAddr]
	p.mu.RUnlock()

	client := p.fetchClient(clients)

	if client == nil {
		return nil, fmt.Errorf("No connections available")
	}

	return client.Subscribe(ctx, req)
}

func (p *Pool) ContainerMetrics(dopplerAddr string, ctx context.Context, req *plumbing.ContainerMetricsRequest) (*plumbing.ContainerMetricsResponse, error) {
	p.mu.RLock()
	clients := p.dopplers[dopplerAddr]
	p.mu.RUnlock()

	client := p.fetchClient(clients)

	if client == nil {
		return nil, fmt.Errorf("No connections available")
	}

	return client.ContainerMetrics(ctx, req)
}

func (p *Pool) RecentLogs(dopplerAddr string, ctx context.Context, req *plumbing.RecentLogsRequest) (*plumbing.RecentLogsResponse, error) {
	p.mu.RLock()
	clients := p.dopplers[dopplerAddr]
	p.mu.RUnlock()

	client := p.fetchClient(clients)

	if client == nil {
		return nil, fmt.Errorf("No connections available")
	}

	return client.RecentLogs(ctx, req)
}

func (p *Pool) Close(dopplerAddr string) {
	p.mu.RLock()
	clients := p.dopplers[dopplerAddr]
	delete(p.dopplers, dopplerAddr)
	p.mu.RUnlock()

	for i := range clients {
		clt := atomic.LoadPointer(&clients[i])
		if clt == nil ||
			(*clientInfo)(clt) == nil {
			continue
		}

		client := *(*clientInfo)(clt)
		client.closer.Close()
	}
}

func (p *Pool) fetchClient(clients []unsafe.Pointer) plumbing.DopplerClient {
	for i := range clients {
		idx := (i + rand.Int()) % p.size
		clt := atomic.LoadPointer(&clients[idx])
		if clt == nil ||
			(*clientInfo)(clt) == nil {
			continue
		}

		client := *(*clientInfo)(clt)
		return client.client
	}

	return nil
}

func (p *Pool) connectToDoppler(addr string, clients []unsafe.Pointer, idx int) {
	for {
		log.Printf("Adding doppler %s", addr)
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			// TODO: We don't yet understand how this could happen, we should.
			// TODO: Replace with exponential backoff.
			log.Printf("Unable to Dial doppler %s: %s", addr, err)
			time.Sleep(5 * time.Second)
			continue
		}

		client := plumbing.NewDopplerClient(conn)
		info := clientInfo{
			client: client,
			closer: conn,
		}

		atomic.StorePointer(&clients[idx], unsafe.Pointer(&info))
		return
	}
}
