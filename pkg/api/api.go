/*
Copyright paskal.maksim@gmail.com
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package api

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/maksim-paskal/envoy-sidecar-helper/pkg/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	envoyHost          = flag.String("envoy.host", "http://127.0.0.1", "envoy host")
	envoyPort          = flag.Int("envoy.port", 18000, "envoy port")
	podName            = flag.String("pod", os.Getenv("POD_NAME"), "pod name")
	namespace          = flag.String("namespace", os.Getenv("POD_NAMESPACE"), "namespace")
	containersName     = flag.String("container", "", "container or containers to watch (splited with comma)")
	envoyReadyCheck    = flag.Bool("envoy.ready.check", true, "check envoy is ready")
	envoyReadyFile     = flag.String("envoy.ready.file", "/envoy-sidecar-helper/envoy.ready", "")
	envoyReadyEndpoint = flag.String("envoy.endpoint.ready", "/ready", "endpoint to check envoy is ready")
	envoyQuitEndpoint  = flag.String("envoy.endpoint.quit", "/quitquitquit", "endpoint to quit envoy")
	checkDuration      = flag.Duration("check.duration", time.Second, "duration to check if container is stopped")
	httpTimeout        = flag.Duration("http.timeout", time.Second*5, "http timeout")
)

var httpClient = &http.Client{
	Timeout: *httpTimeout,
}

var ctx = context.Background()

var (
	errContainerStillRunning = errors.New("containers still running")
	errStatusNotOK           = errors.New("envoy return not OK status")
)

// wait for envoy sidecar to be ready.
func CheckEnvoyStart() {
	if !*envoyReadyCheck {
		log.Info("envoy ready check disabled")

		return
	}

	log.Infof("waiting for envoy will be ready %s:%d", *envoyHost, *envoyPort)

	for {
		time.Sleep(*checkDuration)

		if err := makeCall("GET", *envoyReadyEndpoint); err != nil {
			log.WithError(err).Debug()
		} else {
			break
		}
	}

	log.Info("envoy is ready")

	if err := os.WriteFile(*envoyReadyFile, []byte("ok"), 0o644); err != nil { //nolint:gosec
		log.WithError(err).Error()
	}
}

// check if container is stoped.
func IsContainerStoped() (bool, error) {
	pod, err := client.KubeClient().CoreV1().Pods(*namespace).Get(ctx, *podName, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrap(err, "error getting pod")
	}

	// use first container if not specified
	podContainersName := []string{pod.Spec.Containers[0].Name}

	// if container name is specified, use it
	if len(*containersName) > 0 {
		podContainersName = strings.Split(*containersName, ",")
	}

	log.Debugf("containers to watch %v", podContainersName)

	foundContainers := 0

	// check if containers is stopped
	for _, containerStatus := range pod.Status.ContainerStatuses {
		for _, podContainerName := range podContainersName {
			if containerStatus.Name == podContainerName {
				if containerStatus.State.Terminated != nil {
					foundContainers++
				}
			}
		}
	}

	log.Debugf("foundContainers=%d,allPodContainers=%d", foundContainers, len(podContainersName))

	// if all watched containers are stopped, return true
	if foundContainers == len(podContainersName) {
		return true, nil
	}

	containersName := strings.Join(podContainersName, ",")

	// if not all watched containers are stopped, return false
	return false, errors.Wrap(errContainerStillRunning, containersName)
}

// check is watched containers is stopped.
func CheckContainerStop() {
	log.Info("waiting for container stop")

	for {
		time.Sleep(*checkDuration)

		stoped, err := IsContainerStoped()
		if err != nil {
			log.WithError(err).Error()
		}

		if stoped {
			break
		}
	}

	if err := makeCall("POST", *envoyQuitEndpoint); err != nil {
		log.WithError(err).Error()
	}
}

// make http call to envoy sidecar.
func makeCall(method string, path string) error {
	ctx := context.Background()

	url := fmt.Sprintf("%s:%d%s", *envoyHost, *envoyPort, path)

	log.Debugf("create request %s, %s", method, url)

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return errors.Wrap(err, "error in http.NewRequestWithContext")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "errors in httpClient.Do")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(errStatusNotOK, "response status not OK")
	}

	return nil
}
