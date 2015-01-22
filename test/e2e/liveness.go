/*
Copyright 2015 Google Inc. All rights reserved.

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

// Tests for liveness probes, both with http and with docker exec.
// These tests use the descriptions in examples/liveness to create test pods.

package e2e

import (
	"time"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/golang/glog"
)

func runLivenessTest(c *client.Client, yamlFileName string) bool {
	// Read the pod description from the YAML file.
	podDescr := loadPodOrDie(assetPath("examples", "liveness", yamlFileName))
	// Randomize the pod name to prevent the test to fail due to problems in clean up from
	// previous tests or parallel executions of this same test.
	podName := podDescr.Name + "-" + string(util.NewUUID())
	podDescr.Name = podName
	// Create the pod.
	glog.Infof("Creating pod %s", podName)
	_, err := c.Pods(api.NamespaceDefault).Create(podDescr)
	if err != nil {
		glog.Infof("Failed to create pod %s: %v", podName, err)
		return false
	}
	// At the end of the test, clean up by removing the pod.
	defer c.Pods(api.NamespaceDefault).Delete(podName)
	// Wait until the pod is not pending. (Here we need to check for something other than
	// 'Pending' other than checking for 'Running', since when failures occur, we go to
	// 'Terminated' which can cause indefinite blocking.)
	if !waitForPodNotPending(c, podName) {
		glog.Infof("Failed to start pod %s", podName)
		return false
	}
	glog.Infof("Started pod %s", podName)

	// Check the pod's current state and verify that restartCount is present.
	pod, err := c.Pods(api.NamespaceDefault).Get(podName)
	if err != nil {
		glog.Errorf("Get pod %s failed: %v", podName, err)
		return false
	}
	initialRestartCount := pod.Status.Info["liveness"].RestartCount
	glog.Infof("Initial restart count of pod %s is %d", podName, initialRestartCount)

	// Wait for at most 48 * 5 = 240s = 4 minutes until restartCount is incremented
	for i := 0; i < 48; i++ {
		// Wait until restartCount is incremented.
		time.Sleep(5 * time.Second)
		pod, err = c.Pods(api.NamespaceDefault).Get(podName)
		if err != nil {
			glog.Errorf("Get pod %s failed: %v", podName, err)
			return false
		}
		restartCount := pod.Status.Info["liveness"].RestartCount
		glog.Infof("Restart count of pod %s is now %d", podName, restartCount)
		if restartCount > initialRestartCount {
			glog.Infof("Restart count of pod %s increased from %d to %d during the test", podName, initialRestartCount, restartCount)
			return true
		}
	}

	glog.Errorf("Did not see the restart count of pod %s increase from %d during the test", podName, initialRestartCount)
	return false
}

// TestLivenessHttp tests restarts with a /healthz http liveness probe.
func TestLivenessHttp(c *client.Client) bool {
	return runLivenessTest(c, "http-liveness.yaml")
}

// TestLivenessExec tests restarts with a docker exec "cat /tmp/health" liveness probe.
func TestLivenessExec(c *client.Client) bool {
	return runLivenessTest(c, "exec-liveness.yaml")
}
