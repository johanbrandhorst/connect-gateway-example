package server

import (
	"context"
	"sync"

	connect "github.com/bufbuild/connect-go"
	"github.com/gofrs/uuid"
	usersv1 "github.com/johanbrandhorst/connect-gateway-example/proto/users/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Backend implements the protobuf interface
type Backend struct {
	mu    *sync.RWMutex
	users []*usersv1.User
}

// New initializes a new Backend struct.
func New() *Backend {
	return &Backend{
		mu: &sync.RWMutex{},
	}
}

// AddUser adds a user to the in-memory store.
func (b *Backend) AddUser(ctx context.Context, req *connect.Request[usersv1.AddUserRequest]) (*connect.Response[usersv1.User], error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	user := &usersv1.User{
		Id:         uuid.Must(uuid.NewV4()).String(),
		Email:      req.Msg.GetEmail(),
		CreateTime: timestamppb.Now(),
	}
	b.users = append(b.users, user)

	return connect.NewResponse(user), nil
}

// GetUser gets a user from the store.
func (b *Backend) GetUser(ctx context.Context, req *connect.Request[usersv1.GetUserRequest]) (*connect.Response[usersv1.User], error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, user := range b.users {
		if user.Id == req.Msg.GetId() {
			return connect.NewResponse(user), nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "user with ID %q could not be found", req.Msg.GetId())
}

// ListUsers lists all users in the store.
func (b *Backend) ListUsers(ctx context.Context, req *connect.Request[usersv1.ListUsersRequest], srv *connect.ServerStream[usersv1.User]) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, user := range b.users {
		err := srv.Send(user)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateUser updates properties on a user.
func (b *Backend) UpdateUser(ctx context.Context, req *connect.Request[usersv1.UpdateUserRequest]) (*connect.Response[usersv1.User], error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var u *usersv1.User
	for _, user := range b.users {
		if user.Id == req.Msg.GetUser().GetId() {
			u = user
			break
		}
	}
	if u == nil {
		return nil, status.Errorf(codes.NotFound, "user with ID %q could not be found", req.Msg.GetUser().GetId())
	}

	for _, path := range req.Msg.GetUpdateMask().GetPaths() {
		switch path {
		case "email":
			u.Email = req.Msg.GetUser().GetEmail()
		default:
			return nil, status.Errorf(codes.InvalidArgument, "cannot update field %q on user", path)
		}
	}

	return connect.NewResponse(u), nil
}
