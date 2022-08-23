package podstate_test

import (
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/appuio/scheduler-canary-controller/controllers/podstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestStatesOverLifecycle(t *testing.T) {
	pods := readPodFixtures(t)

	assert.Equal(t, podstate.PodCreated, podstate.State(pods["pod_state_0"]))
	assert.Equal(t, podstate.PodScheduled, podstate.State(pods["pod_state_1"]))
	assert.Equal(t, podstate.PodAcknowledged, podstate.State(pods["pod_state_2"]))
	assert.Equal(t, podstate.PodWaiting, podstate.State(pods["pod_state_3"]))
	assert.Equal(t, podstate.PodRunning, podstate.State(pods["pod_state_4"]))
	assert.Equal(t, podstate.PodRunning, podstate.State(pods["pod_state_5"]))
	assert.Equal(t, podstate.PodCompleted, podstate.State(pods["pod_state_6"]))
}

func readPodFixtures(t *testing.T) map[string]corev1.Pod {
	pods := make(map[string]corev1.Pod)

	td := os.DirFS("testdata")
	files, err := fs.ReadDir(td, ".")
	require.NoError(t, err)
	for _, f := range files {
		if !f.Type().IsRegular() {
			continue
		}
		r, err := fs.ReadFile(td, f.Name())
		require.NoError(t, err)
		var pod corev1.Pod
		require.NoError(t, yaml.Unmarshal(r, &pod))
		pods[strings.TrimSuffix(f.Name(), ".yml")] = pod
	}

	return pods
}
