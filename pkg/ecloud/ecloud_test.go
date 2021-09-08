package ecloud

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var (
	testHost      = ""
	testCluster   = ""
	testAccessKey = ""
	testSecretKey = ""
)

func TestMain(m *testing.M) {
	testHost = os.Getenv("ECLOUD_HOST")
	testCluster = os.Getenv("ECLOUD_CLUSTER")
	testAccessKey = os.Getenv("ECLOUD_ACCESS_KEY")
	testSecretKey = os.Getenv("ECLOUD_SECRET_KEY")

	os.Exit(m.Run())
}

func TestEcloudClientAddNode(t *testing.T) {
	logger := logrus.New()
	ec, err := NewEcloudClient(testHost, testAccessKey, testSecretKey)
	assert.NoError(t, err)
	ctx := context.Background()
	request := &VMInfrastructure{
		ServerType:   "VM",
		ServerVmType: "common",
		CPU:          2,
		Disk:         20,
		Ram:          4,
		ImageId:      "Image For KCS_V7.5",
		Volumes: VMVolumes{
			SystemDisk: VMVolume{
				Size:       50,
				VolumeType: "performanceOptimization",
			},
			DataDisk: VMVolume{
				Size:       50,
				VolumeType: "ebs_ceph_cache",
			},
		},
		Keypair:      "hexintest",
		SpecsName:    "s1.large.2",
		MaxBandWidth: "1",
	}
	err = ec.AddNode(ctx, logger, testCluster, request)
	assert.NoError(t, err, err)
}

func TestEcloudClientNodes(t *testing.T) {
	logger := logrus.New()
	ec, err := NewEcloudClient(testHost, testAccessKey, testSecretKey)
	assert.NoError(t, err)
	ctx := context.Background()

	nl, err := ec.Nodes(ctx, logger, testCluster)
	assert.NoError(t, err)
	assert.NotNil(t, nl)
	assert.Greater(t, nl.Total, 0)
}
