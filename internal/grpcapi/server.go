// SPDX-License-Identifier: Apache-2.0

package grpcapi

import (
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	pb "github.com/complytime-labs/complytime-core/gen/complytime/v1"
	"github.com/complytime-labs/complytime-core/internal/store"
)

// GRPCServer wraps grpc.Server for use by the gateway main function.
type GRPCServer struct {
	inner *grpc.Server
}

// Serve starts serving on the given listener (blocking).
func (s *GRPCServer) Serve(lis net.Listener) error {
	return s.inner.Serve(lis)
}

// GracefulStop stops the gRPC server gracefully.
func (s *GRPCServer) GracefulStop() {
	s.inner.GracefulStop()
}

// NewServer creates a gRPC server with PolicyService registered.
// Additional services (Evidence, Audit, Catalog) are added here
// as they are implemented.
func NewServer(s store.Stores) *GRPCServer {
	srv := grpc.NewServer()

	pb.RegisterPolicyServiceServer(srv, NewPolicyServer(s.Policies, s.Mappings))

	hsrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, hsrv)
	hsrv.SetServingStatus("complytime.v1.PolicyService", healthpb.HealthCheckResponse_SERVING)

	reflection.Register(srv)

	return &GRPCServer{inner: srv}
}
