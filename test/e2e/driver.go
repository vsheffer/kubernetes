/*
Copyright 2014 Google Inc. All rights reserved.

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

package e2e

import (
	"math/rand"
	"time"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/golang/glog"
)

type testSpec struct {
	// The test to run
	test func(c *client.Client) bool
	// The human readable name of this test
	name string
}

type testInfo struct {
	passed bool
	spec   testSpec
}

// Output a summary in the TAP (test anything protocol) format for automated processing.
// See http://testanything.org/ for more info
func outputTAPSummary(infoList []testInfo) {
	glog.Infof("1..%d", len(infoList))
	for i, info := range infoList {
		if info.passed {
			glog.Infof("ok %d - %s", i+1, info.spec.name)
		} else {
			glog.Infof("not ok %d - %s", i+1, info.spec.name)
		}
	}
}

// Fisher-Yates shuffle using the given RNG r
func shuffleTests(tests []testSpec, r *rand.Rand) {
	for i := len(tests) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		tests[i], tests[j] = tests[j], tests[i]
	}
}

// Run each Go end-to-end-test. This function assumes the
// creation of a test cluster.
func RunE2ETests(authConfig, certDir, host, repoRoot, provider string, orderseed int64, times int, testList []string) {
	testContext = testContextType{authConfig, certDir, host, repoRoot, provider}
	util.ReallyCrash = true
	util.InitLogs()
	defer util.FlushLogs()

	// TODO: Associate a timeout with each test individually.
	go func() {
		defer util.FlushLogs()
		time.Sleep(10 * time.Minute)
		glog.Fatalf("This test has timed out. Cleanup not guaranteed.")
	}()

	tests := []testSpec{
		/*  Disable TestKubernetesROService due to rate limiter issues.
		    TODO: Add this test back when rate limiting is working properly.
				{TestKubernetesROService, "TestKubernetesROService"},
		*/
		{TestKubeletSendsEvent, "TestKubeletSendsEvent"},
		{TestImportantURLs, "TestImportantURLs"},
		{TestPodUpdate, "TestPodUpdate"},
		{TestNetwork, "TestNetwork"},
		{TestClusterDNS, "TestClusterDNS"},
		{TestPodHasServiceEnvVars, "TestPodHasServiceEnvVars"},
		{TestBasic, "TestBasic"},
		{TestPrivate, "TestPrivate"},
		{TestLivenessHttp, "TestLivenessHttp"},
		{TestLivenessExec, "TestLivenessExec"},
	}

	// Check testList for non-existent tests and populate a StringSet with tests to run.
	validTestNames := util.NewStringSet()
	for _, test := range tests {
		validTestNames.Insert(test.name)
	}
	runTestNames := util.NewStringSet()
	for _, testName := range testList {
		if validTestNames.Has(testName) {
			runTestNames.Insert(testName)
		} else {
			glog.Warningf("Requested test %s does not exist", testName)
		}
	}

	// if testList was specified, filter down now before we expand and shuffle
	if len(testList) > 0 {
		newTests := make([]testSpec, 0)
		for i, test := range tests {
			// Check if this test is supposed to run, either if listed explicitly in
			// a --test flag or if no --test flags were supplied.
			if !runTestNames.Has(test.name) {
				glog.Infof("Skipping test %d %s", i+1, test.name)
				continue
			}
			newTests = append(newTests, test)
		}
		tests = newTests
	}
	if times != 1 {
		newTests := make([]testSpec, 0, times*len(tests))
		for i := 0; i < times; i++ {
			newTests = append(newTests, tests...)
		}
		tests = newTests
	}
	if orderseed == 0 {
		// Use low order bits of NanoTime as the default seed. (Using
		// all the bits makes for a long, very similar looking seed
		// between runs.)
		orderseed = time.Now().UnixNano() & (1<<32 - 1)
	}
	// TODO(satnam6502): When the tests are run in parallel we will
	//                   no longer need the shuffle.
	shuffleTests(tests, rand.New(rand.NewSource(orderseed)))
	glog.Infof("Tests shuffled with orderseed %#x\n", orderseed)

	info := []testInfo{}
	passed := true
	for i, test := range tests {
		glog.Infof("Running test %d %s", i+1, test.name)
		// A client is made for each test. This allows us to attribute
		// issues with rate ACLs etc. to a specific test and supports
		// parallel testing.
		testPassed := test.test(loadClientOrDie())
		if !testPassed {
			glog.Infof("        test %d failed", i+1)
			passed = false
		} else {
			glog.Infof("        test %d passed", i+1)
		}
		// TODO: clean up objects created during a test after the test, so cases
		// are independent.
		info = append(info, testInfo{testPassed, test})
	}
	outputTAPSummary(info)
	if !passed {
		glog.Fatalf("At least one test failed")
	} else {
		glog.Infof("All tests pass")
	}
}
