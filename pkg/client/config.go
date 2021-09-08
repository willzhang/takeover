package client

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/hexinatgithub/takeover/pkg/ecloud"
)

func valideVMTemplate(c *ecloud.VMTemplate) error {
	if c.ClusterID == "" {
		return errors.New("cluster_id is empty")
	}

	if c.Infrastructure.ServerVmType == "" {
		return errors.New("infrastructure.serverVmType is empty, value range ['common', 'memImprove']")
	}

	if c.Infrastructure.CPU == 0 {
		return errors.New("infrastructure.cpu value is 0")
	}

	if c.Infrastructure.Ram == 0 {
		return errors.New("infrastructure.ram value is 0")
	}

	if c.Infrastructure.Disk == 0 {
		return errors.New("infrastructure.disk value is 0")
	}

	if c.Infrastructure.Password != "" && c.Infrastructure.Keypair != "" {
		return errors.New("only password or keypair can be spec, but not both")
	}

	if c.Infrastructure.Volumes.SystemDisk.Size == 0 {
		return errors.New("infrastructure.Volumes.systemDisk.size value is 0, must greater than 20")
	}
	if c.Infrastructure.Volumes.SystemDisk.VolumeType == "" {
		return errors.New("infrastructure.Volumes.systemDisk.volumeType is empty, value range [highPerformance, performanceOptimization]")
	}

	if c.Infrastructure.Volumes.DataDisk.Size == 0 {
		return errors.New("infrastructure.Volumes.dataDisk.size value is 0, must greater than 20")
	}
	if c.Infrastructure.Volumes.DataDisk.VolumeType == "" {
		return errors.New("infrastructure.Volumes.dataDisk.volumeType is empty, value range [highPerformance, performanceOptimization]")
	}

	if c.Infrastructure.SpecsName == "" {
		return errors.New("infrastructure.specsname is empty")
	}

	return nil
}

func completeVMTemplate(c *ecloud.VMTemplate) {
	if c.Infrastructure.ServerVmType == "" {
		c.Infrastructure.ServerVmType = "common"
	}
}

var (
	defaultIncreaseNum = 2
)

type VMAutoScale struct {
	IncreaseNum *int              `json:"increase_num"`
	VMTemplate  ecloud.VMTemplate `json:"vm_template"`
}

func (s *VMAutoScale) SetIncreaseNum(n int) {
	s.IncreaseNum = &n
}

func valideVMAutoScale(s *VMAutoScale) error {
	if s.IncreaseNum == nil {
		return errors.New("increase_num must spec")
	}
	return valideVMTemplate(&s.VMTemplate)
}

func completeVMAutoScale(s *VMAutoScale) {
	if s.IncreaseNum == nil {
		s.IncreaseNum = &defaultIncreaseNum
	}
	completeVMTemplate(&s.VMTemplate)
}

type TakeroverConfig struct {
	Host        string      `json:"host"`
	AccessKey   string      `json:"access_key"`
	SecretKey   string      `json:"secret_key"`
	VMAutoScale VMAutoScale `json:"vm_auto_scale"`
}

func valideTakeoverConfig(c *TakeroverConfig) error {
	if c.AccessKey == "" {
		return errors.New("access_key is empty")
	}
	if c.SecretKey == "" {
		return errors.New("secret_key is empty")
	}
	return valideVMAutoScale(&c.VMAutoScale)
}

func completeTakeoverConfig(c *TakeroverConfig) {
	completeVMAutoScale(&c.VMAutoScale)
}

func LoadConfig() (*TakeroverConfig, error) {
	filename := configFilePath()

	_, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c := new(TakeroverConfig)
	if err := json.NewDecoder(f).Decode(c); err != nil {
		return nil, err
	}
	completeTakeoverConfig(c)

	if err := valideTakeoverConfig(c); err != nil {
		return nil, err
	}

	return c, nil
}

func configFilePath() string {
	return filepath.Join(os.Getenv("HOME"), ".takeover", "config", "config.json")
}
