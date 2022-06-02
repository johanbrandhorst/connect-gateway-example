package main

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/johanbrandhorst/connect-gateway-example/proto/users/v1/usersv1connect"
	usersv1 "github.com/johanbrandhorst/connect-gateway-example/proto/users/v1/usersv1gateway"
	"github.com/johanbrandhorst/connect-gateway-example/server"
	"github.com/johanbrandhorst/connect-gateway-example/third_party"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

// getOpenAPIHandler serves an OpenAPI UI.
// Adapted from https://github.com/philips/grpc-gateway-example/blob/a269bcb5931ca92be0ceae6130ac27ae89582ecc/cmd/serve.go#L63
func getOpenAPIHandler() http.Handler {
	mime.AddExtensionType(".svg", "image/svg+xml")
	// Use subdirectory in embedded files
	subFS, err := fs.Sub(third_party.OpenAPI, "OpenAPI")
	if err != nil {
		panic("couldn't create sub filesystem: " + err.Error())
	}
	return http.FileServer(http.FS(subFS))
}

func run() error {
	log := grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard)
	grpclog.SetLoggerV2(log)

	addr := "0.0.0.0:8080"
	// Note: this will succeed asynchronously, once we've started the server below.
	conn, err := grpc.DialContext(
		context.Background(),
		"dns:///"+addr,
		grpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}

	gwmux := runtime.NewServeMux(
		runtime.WithMarshalerOption("*", &runtime.HTTPBodyMarshaler{
			Marshaler: &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{UseProtoNames: true},
			},
		}),
	)
	err = usersv1.RegisterUserServiceHandler(context.Background(), gwmux, conn)
	if err != nil {
		return fmt.Errorf("failed to register gateway: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", getOpenAPIHandler())
	mux.Handle(usersv1connect.NewUserServiceHandler(server.New()))
	mux.Handle("/api/v1/", gwmux)
	server := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	log.Info("Serving Connect, gRPC-Gateway and OpenAPI Documentation on http://", addr)
	return server.ListenAndServe()
}
