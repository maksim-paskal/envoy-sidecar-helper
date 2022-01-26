package api

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
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
	containerName  = flag.String("container", "", "container to watch")
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

	podContainerName := *containerName

	// use first container if not specified
	if len(podContainerName) == 0 {
		podContainerName = pod.Spec.Containers[0].Name
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == podContainerName {
			if containerStatus.State.Terminated != nil {
				return true, nil
			}

			return false, nil
		}
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
