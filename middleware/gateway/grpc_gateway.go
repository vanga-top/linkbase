package gateway

import (
	"context"
	"fmt"
	gwruntime "github.com/linkbase/middleware/gateway/runtime"
	"github.com/linkbase/middleware/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"net/http"
)

type Endpoint struct {
	Network, Addr string
}

type Options struct {
	Addr       string
	GRPCServer Endpoint
	OpenAPIDir string
	Mux        []gwruntime.ServeMuxOption
}

func Run(ctx context.Context, opts Options) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	conn, err := dial(ctx, opts.GRPCServer.Network, opts.GRPCServer.Addr)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		if err := conn.Close(); err != nil {
			log.Error("Failed to close a client connection to the gRPC server: %v", zap.Error(err))
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/openapiv2/", openAPIServer(opts.OpenAPIDir))

	return nil
}

func dial(ctx context.Context, network, addr string) (*grpc.ClientConn, error) {
	switch network {
	case "tcp":
		return dialTCP(ctx, addr)
	case "unix":
		return dialUnix(ctx, addr)
	default:
		return nil, fmt.Errorf("unsupported network type %q", network)
	}
}

// dialTCP creates a client connection via TCP.
// "addr" must be a valid TCP address with a port number.
func dialTCP(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

// dialUnix creates a client connection via a unix domain socket.
// "addr" must be a valid path to the socket.
func dialUnix(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	d := func(ctx context.Context, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", addr)
	}
	return grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(d))
}
