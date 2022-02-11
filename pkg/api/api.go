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
	envoyHost      = flag.String("envoy.host", "http://127.0.0.1", "envoy host")
	envoyPort      = flag.Int("envoy.port", 18000, "envoy port")
	podName        = flag.String("pod", os.Getenv("POD_NAME"), "pod name")
	namespace      = flag.String("namespace", os.Getenv("POD_NAMESPACE"), "namespace")
	containersName = flag.String("container", "", "containers to watch, may be splited by comma")
	envoyReadyFile = flag.String("envoy.readyFile", "/envoy-sidecar-helper/envoy.ready", "")
)

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

var ctx = context.Background()

var (
	errContainerNotFound = errors.New("container not found")
	errStatusNotOK       = errors.New("envoy return not OK status")
)

func CheckEnvoyStart() {
	log.Infof("waiting for envoy will be ready %s:%d", *envoyHost, *envoyPort)

	for {
		time.Sleep(time.Second)

		if err := makeCall("GET", "/ready"); err != nil {
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

func IsContainerStoped() (bool, error) {
	pod, err := client.KubeClient().CoreV1().Pods(*namespace).Get(ctx, *podName, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrap(err, "error getting pod")
	}

	// use first container if not specified
	podContainersName := []string{pod.Spec.Containers[0].Name}

	if len(*containersName) > 0 {
		podContainersName = strings.Split(*containersName, ",")
	}

	log.Debugf("containers to watch %v", podContainersName)

	foundContainers := 0

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

	if foundContainers == len(podContainersName) {
		return true, nil
	}

	return false, errContainerNotFound
}

func CheckContainerStop() {
	log.Info("waiting for container stop")

	for {
		time.Sleep(time.Second)

		stoped, err := IsContainerStoped()
		if err != nil {
			log.WithError(err).Error()
		}

		if stoped {
			break
		}
	}

	if err := makeCall("POST", "/quitquitquit"); err != nil {
		log.WithError(err).Error()
	}
}

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
		return errStatusNotOK
	}

	return nil
}
