package ecloud

import (
	"context"

	"github.com/sirupsen/logrus"
)

const (
	defaultIncreaseNum = 2
)

type VMTemplate struct {
	ClusterID      string           `json:"cluster_id"`
	Infrastructure VMInfrastructure `json:"infrastructure"`
}

type Scaler interface {
	Nodes(ctx context.Context, logger *logrus.Logger) (*NodeList, error)
	AddNode(ctx context.Context, logger *logrus.Logger) error
	Grow(ctx context.Context, logger *logrus.Logger) error
}

type scaler struct {
	increaseNum  int
	template     VMTemplate
	ecloudClient *EcloudClient
}

func NewScaler(
	increaseNum int,
	template VMTemplate,
	ecloudClient *EcloudClient,
) Scaler {
	if increaseNum < 2 {
		increaseNum = defaultIncreaseNum
	}
	return &scaler{
		increaseNum:  increaseNum,
		template:     template,
		ecloudClient: ecloudClient,
	}
}

func (s *scaler) calculate(num int) int {
	return num + s.increaseNum
}

func (s *scaler) Grow(ctx context.Context, logger *logrus.Logger) error {
	nodeList, err := s.Nodes(ctx, logger)
	if err != nil {
		return err
	}

	nt := s.calculate(nodeList.Total)
	logger.Infof("grow disaster kubernetes cluster, current nodes %d, need to add %d nodes", nodeList.Total, s.increaseNum)
	for i := 1; i <= s.increaseNum; i++ {
		if err := s.AddNode(ctx, logger); err != nil {
			logger.Errorf("add nodes error, add %d nodes succeed, left %d nodes not added, abort grow process", i-1, s.increaseNum-i+1)
			return err
		}
	}
	logger.Infof("disaster kubernetes cluster grow from %d to %d", nodeList.Total, nt)
	return nil
}

func (s *scaler) Nodes(ctx context.Context, logger *logrus.Logger) (*NodeList, error) {
	return s.ecloudClient.Nodes(ctx, logger, s.template.ClusterID)
}

func (s *scaler) AddNode(ctx context.Context, logger *logrus.Logger) error {
	return s.ecloudClient.AddNode(ctx, logger, s.template.ClusterID, &s.template.Infrastructure)
}
