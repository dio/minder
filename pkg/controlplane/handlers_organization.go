// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type createOrganizationValidation struct {
	Name    string `db:"name" validate:"required"`
	Company string `db:"company" validate:"required"`
}

// CreateOrganization is a service for creating an organization
func (s *Server) CreateOrganization(ctx context.Context,
	in *pb.CreateOrganizationRequest) (*pb.CreateOrganizationResponse, error) {
	// validate that the company and name are not empty
	validator := validator.New()
	err := validator.Struct(createOrganizationValidation{Name: in.Name, Company: in.Company})
	if err != nil {
		return nil, err
	}

	org, err := s.store.CreateOrganization(ctx, db.CreateOrganizationParams{Name: in.Name, Company: in.Company})
	if err != nil {
		return nil, err
	}

	response := &pb.CreateOrganizationResponse{Id: org.ID, Name: org.Name,
		Company: org.Company, CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt)}

	if in.CreateDefaultRecords {
		// we need to create the default records for the organization
		protectedPtr := true
		adminPtr := true
		group, _ := s.CreateGroup(ctx, &pb.CreateGroupRequest{
			OrganizationId: org.ID,
			Name:           fmt.Sprintf("%s-admin", org.Name),
			Description:    fmt.Sprintf("Default admin group for %s", org.Name),
			IsProtected:    &protectedPtr,
		})

		if group != nil {
			grp := pb.GroupRecord{GroupId: group.GroupId, OrganizationId: group.OrganizationId,
				Name: group.Name, Description: group.Description,
				IsProtected: group.IsProtected, CreatedAt: group.CreatedAt, UpdatedAt: group.UpdatedAt}
			response.DefaultGroup = &grp

			// we can create the default role for org and for group
			role, _ := s.CreateRoleByOrganization(ctx, &pb.CreateRoleByOrganizationRequest{
				OrganizationId: org.ID,
				Name:           fmt.Sprintf("%s-org-admin", org.Name),
				IsAdmin:        &adminPtr,
				IsProtected:    &protectedPtr,
			})

			roleGroup, _ := s.CreateRoleByGroup(ctx, &pb.CreateRoleByGroupRequest{
				OrganizationId: org.ID,
				GroupId:        group.GroupId,
				Name:           fmt.Sprintf("%s-group-admin", org.Name),
				IsAdmin:        &adminPtr,
				IsProtected:    &protectedPtr,
			})

			if role != nil && roleGroup != nil {
				rl := pb.RoleRecord{Id: role.Id, Name: role.Name, IsAdmin: role.IsAdmin,
					IsProtected: role.IsProtected, CreatedAt: role.CreatedAt, UpdatedAt: role.UpdatedAt}
				response.DefaultRole = &rl

				// we can create the default user
				needsPasswordChange := true
				user, _ := s.CreateUser(ctx, &pb.CreateUserRequest{
					OrganizationId:      org.ID,
					Username:            fmt.Sprintf("%s-admin", org.Name),
					IsProtected:         &protectedPtr,
					NeedsPasswordChange: &needsPasswordChange,
					RoleIds:             []int32{role.Id, roleGroup.Id},
					GroupIds:            []int32{group.GroupId},
				})
				if user != nil {
					usr := pb.UserRecord{Id: user.Id, OrganizationId: org.ID, Username: user.Username,
						Password: user.Password, IsProtected: user.IsProtected, CreatedAt: user.CreatedAt,
						UpdatedAt: user.UpdatedAt, NeedsPasswordChange: user.NeedsPasswordChange}
					response.DefaultUser = &usr
				}
			}
		}
	}

	return response, nil
}

// GetOrganizations is a service for getting a list of organizations
func (s *Server) GetOrganizations(ctx context.Context,
	in *pb.GetOrganizationsRequest) (*pb.GetOrganizationsResponse, error) {

	// define default values for limit and offset
	if in.Limit == nil || *in.Limit == -1 {
		in.Limit = new(int32)
		*in.Limit = PaginationLimit
	}
	if in.Offset == nil {
		in.Offset = new(int32)
		*in.Offset = 0
	}

	orgs, err := s.store.ListOrganizations(ctx, db.ListOrganizationsParams{
		Limit:  *in.Limit,
		Offset: *in.Offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get groups: %s", err)
	}

	var resp pb.GetOrganizationsResponse
	resp.Organizations = make([]*pb.OrganizationRecord, 0, len(orgs))
	for _, org := range orgs {
		resp.Organizations = append(resp.Organizations, &pb.OrganizationRecord{
			Id:        org.ID,
			Name:      org.Name,
			Company:   org.Company,
			CreatedAt: timestamppb.New(org.CreatedAt),
			UpdatedAt: timestamppb.New(org.UpdatedAt),
		})
	}

	return &resp, nil
}

func getOrganizationDependencies(ctx context.Context, store db.Store,
	org db.Organization) ([]*pb.GroupRecord, []*pb.RoleRecord, []*pb.UserRecord, error) {
	const MAX_ITEMS = 999
	// get the groups for the organization
	groups, err := store.ListGroupsByOrganizationID(ctx, org.ID)
	if err != nil {
		return nil, nil, nil, err
	}

	// get the roles for the organization
	roles, err := store.ListRoles(ctx, db.ListRolesParams{OrganizationID: org.ID, Limit: MAX_ITEMS, Offset: 0})
	if err != nil {
		return nil, nil, nil, err
	}

	users, err := store.ListUsersByOrganization(ctx,
		db.ListUsersByOrganizationParams{OrganizationID: org.ID, Limit: MAX_ITEMS, Offset: 0})
	if err != nil {
		return nil, nil, nil, err
	}

	// convert to right data type
	var groupsPB []*pb.GroupRecord
	for _, group := range groups {
		groupsPB = append(groupsPB, &pb.GroupRecord{
			GroupId:        group.ID,
			OrganizationId: group.OrganizationID,
			Name:           group.Name,
			Description:    group.Description.String,
			IsProtected:    group.IsProtected,
			CreatedAt:      timestamppb.New(group.CreatedAt),
			UpdatedAt:      timestamppb.New(group.UpdatedAt),
		})
	}

	var rolesPB []*pb.RoleRecord
	for _, role := range roles {
		rolesPB = append(rolesPB, &pb.RoleRecord{
			Id:             role.ID,
			OrganizationId: role.OrganizationID,
			GroupId:        &role.GroupID.Int32,
			Name:           role.Name,
			IsAdmin:        role.IsAdmin,
			IsProtected:    role.IsProtected,
			CreatedAt:      timestamppb.New(role.CreatedAt),
			UpdatedAt:      timestamppb.New(role.UpdatedAt),
		})
	}

	var usersPB []*pb.UserRecord
	for _, user := range users {
		usersPB = append(usersPB, &pb.UserRecord{
			Id:                  user.ID,
			OrganizationId:      user.OrganizationID,
			Email:               &user.Email.String,
			FirstName:           &user.FirstName.String,
			LastName:            &user.LastName.String,
			Username:            user.Username,
			IsProtected:         &user.IsProtected,
			NeedsPasswordChange: &user.NeedsPasswordChange,
			CreatedAt:           timestamppb.New(user.CreatedAt),
			UpdatedAt:           timestamppb.New(user.UpdatedAt),
		})
	}

	return groupsPB, rolesPB, usersPB, nil
}

// GetOrganization is a service for getting an organization
func (s *Server) GetOrganization(ctx context.Context,
	in *pb.GetOrganizationRequest) (*pb.GetOrganizationResponse, error) {
	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.OrganizationId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	if in.GetOrganizationId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "organization id is required")
	}

	org, err := s.store.GetOrganization(ctx, in.OrganizationId)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get organization: %s", err)
	}

	groups, roles, users, err := getOrganizationDependencies(ctx, s.store, org)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get organization dependencies: %s", err)
	}

	var resp pb.GetOrganizationResponse
	resp.Organization = &pb.OrganizationRecord{
		Id:        org.ID,
		Name:      org.Name,
		Company:   org.Company,
		CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt),
	}
	resp.Groups = groups
	resp.Roles = roles
	resp.Users = users

	return &resp, nil
}

// GetOrganizationByName is a service for getting an organization
func (s *Server) GetOrganizationByName(ctx context.Context,
	in *pb.GetOrganizationByNameRequest) (*pb.GetOrganizationByNameResponse, error) {
	if in.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "organization name is required")
	}

	org, err := s.store.GetOrganizationByName(ctx, in.Name)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get organization: %s", err)
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, org.ID) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	groups, roles, users, err := getOrganizationDependencies(ctx, s.store, org)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get organization dependencies: %s", err)
	}

	var resp pb.GetOrganizationByNameResponse
	resp.Organization = &pb.OrganizationRecord{
		Id:        org.ID,
		Name:      org.Name,
		Company:   org.Company,
		CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt),
	}
	resp.Groups = groups
	resp.Roles = roles
	resp.Users = users

	return &resp, nil
}

type deleteOrganizationValidation struct {
	Id int32 `db:"id" validate:"required"`
}

// DeleteOrganization is a handler that deletes a organization
func (s *Server) DeleteOrganization(ctx context.Context,
	in *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	validator := validator.New()
	err := validator.Struct(deleteOrganizationValidation{Id: in.Id})
	if err != nil {
		return nil, err
	}

	_, err = s.store.GetOrganization(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	if in.Force == nil {
		isProtected := false
		in.Force = &isProtected
	}

	// if we do not force the deletion, we need to check if there are groups
	if !*in.Force {
		// list groups belonging to that organization
		groups, err := s.store.ListGroupsByOrganizationID(ctx, in.Id)
		if err != nil {
			return nil, err
		}

		if len(groups) > 0 {
			errcode := fmt.Errorf("cannot delete the organization, there are groups associated with it")
			return nil, errcode
		}
	}

	// otherwise we delete, and delete groups in cascade
	err = s.store.DeleteOrganization(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteOrganizationResponse{}, nil
}
