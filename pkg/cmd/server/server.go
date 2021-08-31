package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/cmd"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/signals"
	veleroclient "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	"github.com/vmware-tanzu/velero/pkg/util/logging"
	"k8s.io/client-go/kubernetes"

	"github.com/hexinatgithub/takeover/pkg/buildinfo"
	"github.com/hexinatgithub/takeover/pkg/client"
	"github.com/hexinatgithub/takeover/pkg/healthcheck"
	"github.com/hexinatgithub/takeover/pkg/restore"
)

const (
	defaultCheckClusterPeriod = 15 * time.Second
	defaultPort               = 8080
	defaultScheduleName       = "takeover-schedule"
)

type serverConfig struct {
	checkClusterPeriod time.Duration
	formatFlag         *logging.FormatFlag
	scheduleName       string
}

type server struct {
	// kubeClient connect product kubenetes cluster to be health checked
	kubeClient kubernetes.Interface
	// veleroClient connect to the diaster recovery cluster
	veleroClient veleroclient.Interface
	// velero server pod namespace
	namespace string
	// checkClusterPeriod is period to check product kubenetes cluster status
	checkClusterPeriod time.Duration
	// restorer is used for create restore CRD
	restorer restore.Interface
	// healthCheck is used for check product kubenetes cluster health
	healthCheck healthcheck.Interface

	ctx        context.Context
	cancelFunc context.CancelFunc
	logger     *logrus.Logger
}

func NewCommand(f client.Factory) *cobra.Command {
	var (
		logLevelFlag = logging.LogLevelFlag(logrus.InfoLevel)
		config       = &serverConfig{
			checkClusterPeriod: defaultCheckClusterPeriod,
			formatFlag:         logging.NewFormatFlag(),
			scheduleName:       defaultScheduleName,
		}
	)

	command := &cobra.Command{
		Use:   "server",
		Short: "Run the takeover server",
		Long:  "Run the takeover server",
		Run: func(c *cobra.Command, args []string) {
			// go-plugin uses log.Println to log when it's waiting for all plugin processes to complete so we need to
			// set its output to stdout.
			log.SetOutput(os.Stdout)

			logLevel := logLevelFlag.Parse()
			format := config.formatFlag.Parse()

			// Make sure we log to stdout so cloud log dashboards don't show this as an error.
			logrus.SetOutput(os.Stdout)

			// Takeover's DefaultLogger logs to stdout, so all is good there.
			logger := logging.DefaultLogger(logLevel, format)

			logger.Infof("setting log-level to %s", strings.ToUpper(logLevel.String()))

			logger.Infof("Starting takeover server %s (%s)", buildinfo.Version, buildinfo.FormattedGitSHA())

			s, err := newserver(f, config, logger)
			cmd.CheckError(err)

			cmd.CheckError(s.run())
		},
	}

	command.Flags().Var(logLevelFlag, "log-level", fmt.Sprintf("The level at which to log. Valid values are %s.", strings.Join(logLevelFlag.AllowedValues(), ", ")))
	command.Flags().Var(config.formatFlag, "log-format", fmt.Sprintf("The format for log output. Valid values are %s.", strings.Join(config.formatFlag.AllowedValues(), ", ")))
	command.Flags().DurationVar(&config.checkClusterPeriod, "check-cluster-period", defaultCheckClusterPeriod, "check kubenetes health period")
	command.Flags().StringVar(&config.scheduleName, "schedule-name", defaultScheduleName, "product schedule name, use to get backup create by that scheduler which name is schedule-name-[timestamp] format")

	return command
}

func newserver(f client.Factory, config *serverConfig, logger *logrus.Logger) (*server, error) {
	var err error
	s := &server{
		logger:             logger,
		checkClusterPeriod: config.checkClusterPeriod,
		namespace:          f.Namespace(),
	}

	s.kubeClient, err = f.KubeClient()
	if err != nil {
		return nil, err
	}

	s.veleroClient, err = f.Client()
	if err != nil {
		return nil, err
	}

	s.ctx, s.cancelFunc = context.WithCancel(context.TODO())
	s.healthCheck = healthcheck.NewHealthCheck(s.ctx, s.kubeClient, config.checkClusterPeriod, s.logger)
	s.restorer = restore.NewRestorer(s.ctx, s.logger, s.veleroClient, s.namespace, config.scheduleName)

	return s, nil
}

func (s *server) run() error {
	signals.CancelOnShutdown(s.cancelFunc, s.logger)

	logger := s.logger
	disaster := make(chan struct{})
	defer close(disaster)

	go func() {
		for {
			select {
			case status := <-s.healthCheck.Watch():
				if status == healthcheck.StatusDown {
					logger.Infoln("detect product kubenetes cluster is unhealth.")
					disaster <- struct{}{}
				}
			case <-s.ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-disaster:
			logger.Infoln("begin restore kubenetes cluster status.")
			if err := s.restorer.Restore(); err != nil {
				logger.Errorln("restore kubenetes cluster status occur error, abort")
			} else {
				logger.Infoln("restore action is in progress, waiting for next disaster happen...")
			}
		case <-s.ctx.Done():
			logger.Infoln("context canceled, server exit.")
			return nil
		}
	}
}
