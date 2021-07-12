// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package ha

import (
	"context"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/helmit/pkg/kubernetes/core/v1"
	"github.com/onosproject/onos-api/go/onos/kpimon"
	"github.com/onosproject/onos-kpimon/test/utils"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

const (
	onosComponentName = "sd-ran"
)

// GetPodListOrFail gets the list of pods active in the onos-config release. The test is failed if getting the list returns
// an error.
func GetPodListOrFail(t *testing.T) []*v1.Pod {
	release := helm.Chart(onosComponentName).Release(onosComponentName)
	client := kubernetes.NewForReleaseOrDie(release)
	podList, err := client.
		CoreV1().
		Pods().
		List(context.Background())
	assert.NoError(t, err)
	return podList
}

// CrashPodOrFail deletes the given pod and fails the test if there is an error
func CrashPodOrFail(t *testing.T, pod *v1.Pod) {
	err := pod.Delete(context.Background())
	assert.NoError(t, err)
}

// FindPodWithPrefix looks for the first pod whose name matches the given prefix string. The test is failed
// if no matching pod is found.
func FindPodWithPrefix(t *testing.T, prefix string) *v1.Pod {
	podList := GetPodListOrFail(t)
	for _, p := range podList {
		if strings.HasPrefix(p.Name, prefix) {
			return p
		}
	}
	assert.Failf(t, "No pod found matching %s", prefix)
	return nil
}

// GetKPIMonMeasurementsOrFail queries measurement data from onos-kpimon
func GetKPIMonMeasurementsOrFail(t *testing.T) *kpimon.GetResponse {
	var (
		resp *kpimon.GetResponse
		err  error
	)
	client := utils.GetKPIMonClient(t)
	assert.NotNil(t, client)

	req := &kpimon.GetRequest{}

	maxAttempts := 30
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err = client.ListMeasurements(context.Background(), req)
		if err == nil {
			return resp
		}
		time.Sleep(4 * time.Second)
	}

	return nil
}

// TestKPIMonRestart tests that onos-kpimon recovers from crashes
func (s *TestSuite) TestKPIMonRestart(t *testing.T) {
	sim := utils.CreateRanSimulatorWithNameOrDie(t, s.c, "test-kpimon-restart")
	assert.NotNil(t, sim)

	// First make sure that KPIMON came up properly
	resp := GetKPIMonMeasurementsOrFail(t)
	assert.NotNil(t, resp)

	for i := 1; i <= 5; i++ {
		// Crash onos-kpimon
		e2tPod := FindPodWithPrefix(t, "onos-kpimon")
		CrashPodOrFail(t, e2tPod)

		resp = GetKPIMonMeasurementsOrFail(t)
		assert.NotNil(t, resp)
	}

	t.Log("KPM restart test passed")
}
