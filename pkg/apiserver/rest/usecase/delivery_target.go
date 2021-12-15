/*
Copyright 2021 The KubeVela Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package usecase

import (
	"context"
	"errors"

	"github.com/oam-dev/kubevela/pkg/apiserver/datastore"
	"github.com/oam-dev/kubevela/pkg/apiserver/log"
	"github.com/oam-dev/kubevela/pkg/apiserver/model"
	apisv1 "github.com/oam-dev/kubevela/pkg/apiserver/rest/apis/v1"
	"github.com/oam-dev/kubevela/pkg/apiserver/rest/utils/bcode"
)

// DeliveryTargetUsecase deliveryTarget manage api
type DeliveryTargetUsecase interface {
	GetDeliveryTarget(ctx context.Context, deliveryTargetName string) (*model.DeliveryTarget, error)
	DetailDeliveryTarget(ctx context.Context, deliveryTarget *model.DeliveryTarget) (*apisv1.DetailDeliveryTargetResponse, error)
	DeleteDeliveryTarget(ctx context.Context, deliveryTargetName string) error
	CreateDeliveryTarget(ctx context.Context, req apisv1.CreateDeliveryTargetRequest) (*apisv1.DetailDeliveryTargetResponse, error)
	UpdateDeliveryTarget(ctx context.Context, deliveryTarget *model.DeliveryTarget, req apisv1.UpdateDeliveryTargetRequest) (*apisv1.DetailDeliveryTargetResponse, error)
	ListDeliveryTargets(ctx context.Context, page, pageSize int, project string) (*apisv1.ListTargetResponse, error)
}

type deliveryTargetUsecaseImpl struct {
	ds             datastore.DataStore
	projectUsecase ProjectUsecase
}

// NewDeliveryTargetUsecase new DeliveryTarget usecase
func NewDeliveryTargetUsecase(ds datastore.DataStore, projectUsecase ProjectUsecase) DeliveryTargetUsecase {
	return &deliveryTargetUsecaseImpl{
		ds:             ds,
		projectUsecase: projectUsecase,
	}
}

func (dt *deliveryTargetUsecaseImpl) ListDeliveryTargets(ctx context.Context, page, pageSize int, project string) (*apisv1.ListTargetResponse, error) {
	deliveryTarget := model.DeliveryTarget{}
	if project != "" {
		deliveryTarget.Project = project
	}
	deliveryTargets, err := dt.ds.List(ctx, &deliveryTarget, &datastore.ListOptions{Page: page, PageSize: pageSize, SortBy: []datastore.SortOption{{Key: "createTime", Order: datastore.SortOrderDescending}}})
	if err != nil {
		return nil, err
	}

	resp := &apisv1.ListTargetResponse{
		Targets: []apisv1.DeliveryTargetBase{},
	}
	for _, raw := range deliveryTargets {
		target, ok := raw.(*model.DeliveryTarget)
		if ok {
			resp.Targets = append(resp.Targets, *(dt.convertFromDeliveryTargetModel(ctx, target)))
		}
	}
	count, err := dt.ds.Count(ctx, &deliveryTarget, nil)
	if err != nil {
		return nil, err
	}
	resp.Total = count

	return resp, nil
}

// DeleteDeliveryTarget delete application DeliveryTarget
func (dt *deliveryTargetUsecaseImpl) DeleteDeliveryTarget(ctx context.Context, deliveryTargetName string) error {
	deliveryTarget := &model.DeliveryTarget{
		Name: deliveryTargetName,
	}
	if err := dt.ds.Delete(ctx, deliveryTarget); err != nil {
		if errors.Is(err, datastore.ErrRecordNotExist) {
			return bcode.ErrDeliveryTargetNotExist
		}
		return err
	}
	return nil
}

func (dt *deliveryTargetUsecaseImpl) CreateDeliveryTarget(ctx context.Context, req apisv1.CreateDeliveryTargetRequest) (*apisv1.DetailDeliveryTargetResponse, error) {
	deliveryTarget := convertCreateReqToDeliveryTargetModel(req)

	// check deliveryTarget name.
	exit, err := dt.ds.IsExist(ctx, &deliveryTarget)
	if err != nil {
		log.Logger.Errorf("check application name is exist failure %s", err.Error())
		return nil, bcode.ErrDeliveryTargetExist
	}
	if exit {
		return nil, bcode.ErrDeliveryTargetExist
	}
	// check project
	project, err := dt.projectUsecase.GetProject(ctx, req.Project)
	if err != nil {
		return nil, err
	}
	deliveryTarget.Namespace = project.Namespace
	deliveryTarget.Project = project.Name

	if err := dt.ds.Add(ctx, &deliveryTarget); err != nil {
		return nil, err
	}
	return dt.DetailDeliveryTarget(ctx, &deliveryTarget)
}

func (dt *deliveryTargetUsecaseImpl) UpdateDeliveryTarget(ctx context.Context, deliveryTarget *model.DeliveryTarget, req apisv1.UpdateDeliveryTargetRequest) (*apisv1.DetailDeliveryTargetResponse, error) {
	deliveryTargetModel := convertUpdateReqToDeliveryTargetModel(deliveryTarget, req)
	if err := dt.ds.Put(ctx, deliveryTargetModel); err != nil {
		return nil, err
	}
	return dt.DetailDeliveryTarget(ctx, deliveryTargetModel)
}

// DetailDeliveryTarget detail DeliveryTarget
func (dt *deliveryTargetUsecaseImpl) DetailDeliveryTarget(ctx context.Context, deliveryTarget *model.DeliveryTarget) (*apisv1.DetailDeliveryTargetResponse, error) {
	return &apisv1.DetailDeliveryTargetResponse{
		DeliveryTargetBase: *dt.convertFromDeliveryTargetModel(ctx, deliveryTarget),
	}, nil
}

// GetDeliveryTarget get DeliveryTarget model
func (dt *deliveryTargetUsecaseImpl) GetDeliveryTarget(ctx context.Context, deliveryTargetName string) (*model.DeliveryTarget, error) {
	deliveryTarget := &model.DeliveryTarget{
		Name: deliveryTargetName,
	}
	if err := dt.ds.Get(ctx, deliveryTarget); err != nil {
		return nil, err
	}
	return deliveryTarget, nil
}

func convertUpdateReqToDeliveryTargetModel(deliveryTarget *model.DeliveryTarget, req apisv1.UpdateDeliveryTargetRequest) *model.DeliveryTarget {
	deliveryTarget.Alias = req.Alias
	deliveryTarget.Description = req.Description
	deliveryTarget.Cluster = (*model.ClusterTarget)(req.Cluster)
	deliveryTarget.Variable = req.Variable
	return deliveryTarget
}

func convertCreateReqToDeliveryTargetModel(req apisv1.CreateDeliveryTargetRequest) model.DeliveryTarget {
	deliveryTarget := model.DeliveryTarget{
		Name:        req.Name,
		Alias:       req.Alias,
		Description: req.Description,
		Cluster:     (*model.ClusterTarget)(req.Cluster),
		Variable:    req.Variable,
	}
	return deliveryTarget
}

func (dt *deliveryTargetUsecaseImpl) convertFromDeliveryTargetModel(ctx context.Context, deliveryTarget *model.DeliveryTarget) *apisv1.DeliveryTargetBase {
	var appNum int64 = 0
	// TODO: query app num in target
	targetBase := &apisv1.DeliveryTargetBase{
		Name:        deliveryTarget.Name,
		Alias:       deliveryTarget.Alias,
		Description: deliveryTarget.Description,
		Cluster:     (*apisv1.ClusterTarget)(deliveryTarget.Cluster),
		Variable:    deliveryTarget.Variable,
		CreateTime:  deliveryTarget.CreateTime,
		UpdateTime:  deliveryTarget.UpdateTime,
		AppNum:      appNum,
	}

	project, err := dt.projectUsecase.GetProject(ctx, deliveryTarget.Project)
	if err != nil {
		log.Logger.Errorf("query project info failure %s", err.Error())
	}
	if project != nil {
		targetBase.Project = convertProjectModel2Base(project)
	}

	if targetBase.Cluster != nil && targetBase.Cluster.ClusterName != "" {
		cluster, err := _getClusterFromDataStore(ctx, dt.ds, deliveryTarget.Cluster.ClusterName)
		if err != nil {
			log.Logger.Errorf("query cluster info failure %s", err.Error())
		}
		if cluster != nil {
			targetBase.ClusterAlias = cluster.Alias
		}
	}

	return targetBase
}
