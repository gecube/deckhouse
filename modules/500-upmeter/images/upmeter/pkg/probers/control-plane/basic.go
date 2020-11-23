package control_plane

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"upmeter/pkg/app"
	"upmeter/pkg/probe/types"
	"upmeter/pkg/probers/util"
)

/*
CHECK:
Cluster should be able to create and delete a ConfigMap.

Period: 1 minute
Create Namespace timeout: 5 seconds.
Delete Namespace timeout: 60 seconds.
*/
func NewBasicProber() types.Prober {
	var basicProbeRef = types.ProbeRef{
		Group: groupName,
		Probe: "basic-functionality",
	}
	const basicProbePeriod = 5
	const basicProbeTimeout = time.Second * 5

	pr := &types.CommonProbe{
		ProbeRef: &basicProbeRef,
		Period:   basicProbePeriod,
	}

	pr.RunFn = func(start int64) {
		log := pr.LogEntry()

		// Set Unknown result if API server is unavailable
		if !CheckApiAvailable(pr) {
			return
		}

		cmName := util.RandomIdentifier("upmeter-basic")
		cm := &v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: cmName,
				Labels: map[string]string{
					"heritage":      "upmeter",
					"upmeter-agent": util.AgentUniqueId(),
					"upmeter-group": "control-plane",
					"upmeter-probe": "basic",
				},
			},
			Data: map[string]string{
				"key1": "value1",
			},
		}

		if !GarbageCollect(pr, cm.Kind, cm.Labels) {
			return
		}

		util.DoWithTimer(basicProbeTimeout, func() {
			_, err := pr.KubernetesClient.CoreV1().ConfigMaps(app.Namespace).Create(cm)
			if err != nil {
				log.Errorf("Create cm/%s: %v", cmName, err)
				pr.ResultCh <- pr.Result(types.ProbeUnknown)
				return
			}
			err = pr.KubernetesClient.CoreV1().ConfigMaps(app.Namespace).Delete(cm.Name, &metav1.DeleteOptions{})
			if err != nil {
				log.Errorf("Delete cm/%s: %v", cmName, err)
				pr.ResultCh <- pr.Result(types.ProbeFailed)
				return
			}

			if !WaitForObjectDeletion(pr, basicProbeTimeout, cm.Kind, cm.Name) {
				pr.ResultCh <- pr.Result(types.ProbeFailed)
				return
			}

			pr.ResultCh <- pr.Result(types.ProbeSuccess)
		}, func() {
			log.Infof("Exceeds timeout when create/delete cm/%s", cmName)
			pr.ResultCh <- pr.Result(types.ProbeUnknown)
		})

	}

	return pr
}
