package restore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
	apiv1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclient "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Interface interface {
	// Restore find the lastest Backup CRD and use that backup to restore cluster state
	Restore() error
}

type restorer struct {
	ctx          context.Context
	logger       *logrus.Logger
	veleroClient veleroclient.Interface
	namespace    string
	scheduleName string
}

func NewRestorer(
	ctx context.Context,
	logger *logrus.Logger,
	veleroClient veleroclient.Interface,
	namespace,
	scheduleName string,
) Interface {
	return &restorer{
		ctx:          ctx,
		logger:       logger,
		veleroClient: veleroClient,
		namespace:    namespace,
		scheduleName: scheduleName,
	}
}

func (r *restorer) Restore() error {
	backups, err := r.veleroClient.VeleroV1().Backups(r.namespace).List(r.ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", apiv1.ScheduleNameLabel, r.scheduleName)})
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"error": err, "namespace": r.namespace, "scheduleName": r.scheduleName,
		}).Errorf("get schedule backups exception")
		return err
	}
	if len(backups.Items) == 0 {
		r.logger.WithFields(logrus.Fields{"namespace": r.namespace, "scheduleName": r.scheduleName}).
			Errorln("schedule has no backup")
		return errors.New("no backup")
	}

	backup := mostRecentBackup(backups.Items, apiv1.BackupPhaseNew, apiv1.BackupPhaseCompleted, apiv1.BackupPhasePartiallyFailed)
	if backup == nil {
		return errors.New("no backup")
	}

	now := time.Now()
	restore := &apiv1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.namespace,
			Name:      restoreName(now),
			Labels:    restoreLabels(now),
		},
		Spec: apiv1.RestoreSpec{
			BackupName: backup.Name,
		},
	}
	restore, err = r.veleroClient.VeleroV1().Restores(r.namespace).Create(r.ctx, restore, metav1.CreateOptions{})
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"namespace": r.namespace, "name": restoreName(now), "error": err,
		}).Error("create restore error")
		return err
	}
	r.logger.Infof("create %s restore in %s namespace succeed", restore.Name, restore.Namespace)
	return nil
}

func restoreName(t time.Time) string {
	return fmt.Sprintf("takeover-%s", t.Format("20060102150405"))
}

func restoreLabels(t time.Time) map[string]string {
	return map[string]string{
		"takeover-timestamp": t.Format("20060102150405"),
	}
}

// mostRecentBackup returns the backup with the most recent start timestamp that has a phase that's
// in the provided list of allowed phases.
func mostRecentBackup(backups []apiv1.Backup, allowedPhases ...apiv1.BackupPhase) *apiv1.Backup {
	// sort the backups in descending order of start time (i.e. most recent to least recent)
	sort.Slice(backups, func(i, j int) bool {
		// Use .After() because we want descending sort.

		var iStartTime, jStartTime time.Time
		if backups[i].Status.StartTimestamp != nil {
			iStartTime = backups[i].Status.StartTimestamp.Time
		}
		if backups[j].Status.StartTimestamp != nil {
			jStartTime = backups[j].Status.StartTimestamp.Time
		}
		return iStartTime.After(jStartTime)
	})

	// create a map of the allowed phases for easy lookup below
	phases := map[apiv1.BackupPhase]struct{}{}
	for _, phase := range allowedPhases {
		phases[phase] = struct{}{}
	}

	var res *apiv1.Backup
	for i, backup := range backups {
		// if the backup's phase is one of the allowable ones, record
		// the backup and break the loop so we can return it
		if _, ok := phases[backup.Status.Phase]; ok {
			res = &backups[i]
			break
		}
	}

	return res
}
