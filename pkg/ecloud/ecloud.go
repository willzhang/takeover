package ecloud

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Address struct {
	IPVersion string `json:"ipVersion"`
	IPAddress string `json:"ipAddress"`
}

type Node struct {
	NodeID      string            `json:"nodeID"`
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels"`
	CPU         int               `json:"cpu"`
	Memory      int               `json:"memory"`
	Addresses   []Address         `json:"Addresses"`
	Taints      bool              `json:"taints"`
	Schedulable bool              `json:"schedulable"`
	Role        string            `json:"role"`
	Status      string            `json:"status"`
	ProviderID  string            `json:"providerID"`
	CreatedTime string            `json:"createdTime"`
}

type NodeList struct {
	Total int    `json:"total"`
	Nodes []Node `json:"nodes"`
}

type VMVolume struct {
	Size       int    `json:"size"`
	VolumeType string `json:"volumeType"`
}

type VMVolumes struct {
	SystemDisk VMVolume `json:"systemDisk"`
	DataDisk   VMVolume `json:"dataDisk"`
}

type VMInfrastructure struct {
	Flavor       string    `json:"flavor,omitempty"`
	ServerType   string    `json:"serverType,omitempty"`
	ServerVmType string    `json:"serverVmType,omitempty"`
	ImageId      string    `json:"imageId,omitempty"`
	CPU          int       `json:"cpu"`
	Disk         int       `json:"disk"`
	Ram          int       `json:"ram"`
	Password     string    `json:"password,omitempty"`
	Keypair      string    `json:"keypair,omitempty"`
	Volumes      VMVolumes `json:"volumes"`
	SpecsName    string    `json:"SpecsName"`
	MaxBandWidth string    `json:"maxbandwidth,omitempty"`
}

type EcloudClient struct {
	host      string
	accessKey string
	secretKey string
}

func NewEcloudClient(
	host string,
	accessKey string,
	secretKey string,
) (*EcloudClient, error) {
	client := &EcloudClient{
		host:      host,
		accessKey: accessKey,
		secretKey: secretKey,
	}
	if err := valideEcloudClient(client); err != nil {
		return nil, err
	}
	return client, nil
}

type NodeResponse struct {
	RequestID string   `json:"requestId"`
	State     string   `json:"state"`
	Body      NodeList `json:"body"`
}

func (c *EcloudClient) Nodes(
	ctx context.Context, logger *logrus.Logger,
	cluster string,
	pageParam ...int,
) (*NodeList, error) {
	query := url.Values{}
	query.Add("page", "1")
	query.Add("pageSize", "100")

	// set default page and pageSize
	if len(pageParam) >= 2 {
		query.Set("page", fmt.Sprintf("%d", pageParam[0]))
		query.Set("pageSize", fmt.Sprintf("%d", pageParam[1]))
	} else if len(pageParam) == 1 {
		query.Set("page", fmt.Sprintf("%d", pageParam[0]))
	}

	path := fmt.Sprintf("/api/kcs/v2/clusters/%s/nodes", cluster)
	request, err := c.cookHTTPRequest(ctx,
		c.host, path, http.MethodGet,
		query, nil,
	)
	if err != nil {
		logger.WithField("err", err).Errorf("cook HTTP request error")
		return nil, err
	}
	logger.Infof("GET %s", request.URL)

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		logger.WithField("err", err).Error("Nodes error")
		return nil, err
	}
	defer res.Body.Close()

	if err := checkHTTPStatusCode(res); err != nil {
		logger.WithField("err", err).Errorf("http response error")
		return nil, err
	}

	nr := &NodeResponse{}
	if err := json.NewDecoder(res.Body).Decode(nr); err != nil {
		logger.WithField("err", err).Errorf("decode NodeResponse error")
		return nil, err
	}
	logger.Infof("requestId: %s, state: %s, total nodes: %d", nr.RequestID, nr.State, nr.Body.Total)

	return &nr.Body, nil
}

type AddNodeRequest struct {
	AddType        string           `json:"addType"`
	ClusterID      string           `json:"cluster_id"`
	Infrastructure VMInfrastructure `json:"infrastructure"`
}

type AddNodeRes struct {
	RequestID    string `json:"requestId"`
	State        string `json:"state"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

func (c *EcloudClient) AddNode(
	ctx context.Context, logger *logrus.Logger,
	cluster string,
	ifr *VMInfrastructure,
) error {
	playload := &AddNodeRequest{
		AddType:        "new",
		ClusterID:      cluster,
		Infrastructure: *ifr,
	}
	buffer := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buffer).Encode(playload); err != nil {
		return err
	}

	path := fmt.Sprintf("/api/kcs/v2/clusters/%s/nodes", cluster)
	request, err := c.cookHTTPRequest(ctx,
		c.host, path, http.MethodPost,
		nil, buffer,
	)
	if err != nil {
		logger.WithField("err", err).Error("cook http request error")
		return err
	}
	logger.Infof("POST %s", request.URL)

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		logger.WithField("err", err).Error("AddNode error")
		return err
	}
	defer res.Body.Close()

	if err := checkHTTPStatusCode(res); err != nil {
		logger.WithField("err", err).Errorf("http response error")
		return err
	}

	anr := &AddNodeRes{}
	if err := json.NewDecoder(res.Body).Decode(anr); err != nil {
		logger.WithField("err", err).Errorf("decode AddNodeRes error")
		return err
	}
	logger.Infof("AddNodeRes: %v", *anr)
	if anr.State != "OK" {
		return errors.New(anr.ErrorMessage)
	}
	return nil
}

func (c *EcloudClient) cookHTTPRequest(
	ctx context.Context,
	host, path, method string,
	query url.Values,
	body io.Reader,
) (*http.Request, error) {
	if query == nil {
		query = url.Values{}
	}

	request, err := http.NewRequestWithContext(ctx, method, concatenateUrl(host, path), body)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Content-Type", "application/json")
	query.Add("Timestamp", time.Now().Local().Format("2006-01-02T15:04:05Z"))
	query.Add("AccessKey", c.accessKey)
	query.Add("SignatureNonce", uuid.New().String())
	query.Add("SignatureMethod", "HmacSHA1")
	query.Add("SignatureVersion", "V2.0")
	if signature, err := c.doSign(method, path, query); err != nil {
		return nil, err
	} else {
		query.Add("Signature", signature)
	}

	request.URL.RawQuery = query.Encode()
	return request, nil
}

func (c *EcloudClient) doSign(method, path string, query url.Values) (string, error) {
	type queryKey struct {
		key   string
		value []string
	}

	// sort query key
	queryArr := make([]queryKey, 0, len(query))
	for k, v := range query {
		queryArr = append(queryArr, queryKey{k, v})
	}
	sort.Slice(queryArr, func(i, j int) bool {
		return queryArr[i].key < queryArr[j].key
	})
	tmpCanonicalizedQueryArr := []string{}
	for i := range queryArr {
		k := percentEncode(url.QueryEscape(queryArr[i].key))
		v := percentEncode(url.QueryEscape(queryArr[i].value[0]))
		tmpCanonicalizedQueryArr = append(tmpCanonicalizedQueryArr, k+"="+v)
	}
	canonicalizedQueryStr := strings.Join(tmpCanonicalizedQueryArr, "&")

	// generate str to sign
	sha := sha256.New()
	_, err := sha.Write([]byte(canonicalizedQueryStr))
	if err != nil {
		return "", err
	}
	cqssha := hex.EncodeToString(sha.Sum(nil))
	stringToSign := method + "\n" + percentEncode(url.QueryEscape(path)) + "\n" + cqssha

	// generate signature
	key := "BC_SIGNATURE&" + c.secretKey
	hm := hmac.New(sha1.New, []byte(key))
	_, err = hm.Write([]byte(stringToSign))
	if err != nil {
		return "", err
	}
	signature := hex.EncodeToString(hm.Sum(nil))

	return signature, nil
}

func concatenateUrl(host, path string) string {
	return fmt.Sprintf("https://%s%s", host, path)
}

func valideEcloudClient(c *EcloudClient) error {
	if c.host == "" {
		return errors.New("host is empty")
	}

	if c.accessKey == "" {
		return errors.New("access_key is empty")
	}

	if c.secretKey == "" {
		return errors.New("secret_key is empty")
	}

	return nil
}

func percentEncode(s string) string {
	s = strings.ReplaceAll(s, "+", "%20")
	s = strings.ReplaceAll(s, "*", "%2A")
	s = strings.ReplaceAll(s, "%7E", "~")
	return s
}

func checkHTTPStatusCode(res *http.Response) error {
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("http request return %d status code", res.StatusCode)
	}
	return nil
}
