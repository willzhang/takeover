package healthcheck

import (
	"context"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

type Status string

const (
	StatusOK   = Status("OK")
	StatusDown = Status("Down")
)

type Interface interface {
	// Watch will block util status change
	Watch() <-chan Status
}

type healthcheck struct {
	ctx        context.Context
	kubeClient kubernetes.Interface
	preStatus  Status
	period     time.Duration
	logger     *logrus.Logger
	change     chan Status
}

func NewHealthCheck(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	period time.Duration,
	logger *logrus.Logger,
) Interface {
	h := &healthcheck{
		ctx:        ctx,
		kubeClient: kubeClient,
		period:     period,
		logger:     logger,
		change:     make(chan Status),
	}
	h.preStatus = h.getClusterStatus()
	h.periodCheck()
	return h
}

func (h *healthcheck) getClusterStatus() Status {
	result := h.kubeClient.Discovery().RESTClient().Get().AbsPath("/livez").Do(h.ctx)
	if err := result.Error(); err != nil {
		h.logger.WithError(err).Errorln("kubenetes cluster health check failed")
		return StatusDown
	}

	var statusCode int
	result.StatusCode(&statusCode)
	if statusCode != http.StatusOK {
		h.logger.WithField("statusCode", statusCode).Errorln("kubenetes cluster is running in unknown status")
		return StatusDown
	}

	return StatusOK
}

// periodCheck check kubenetes cluster until status changed.
func (h *healthcheck) periodCheck() {
	go func() {
		ticker := time.NewTicker(h.period)
		h.change <- h.preStatus
		for {
			select {
			case <-ticker.C:
				if status := h.getClusterStatus(); status != h.preStatus {
					h.logger.Infof("kubernetes cluster status change from %s to %s", h.preStatus, status)
					h.preStatus = status
					h.change <- status
				} else {
					h.logger.Infof("kubernetes cluster current status %s", status)
				}
			case <-h.ctx.Done():
				close(h.change)
				return
			}
		}
	}()
}

func (h *healthcheck) Watch() <-chan Status {
	return h.change
}
