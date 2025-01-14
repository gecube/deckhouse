/*
Copyright 2021 Flant CJSC
Licensed under the Deckhouse Platform Enterprise Edition (EE) license. See https://github.com/deckhouse/deckhouse/blob/main/ee/LICENSE
*/

package madison

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/sdk"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/deckhouse/deckhouse/go_lib/dependency"
)

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue:        "/modules/flant-integration/connect_registration",
	OnBeforeHelm: &go_hook.OrderedConfig{Order: 20},
}, dependency.WithExternalDependencies(registrationHandler))

const (
	connectBaseURL   = "https://connect.deckhouse.io"
	registrationURL  = connectBaseURL + "/v1/madison_register"
	connectStatusURL = connectBaseURL + "/v1/madison_status"

	madisonKeyPath = "flantIntegration.madisonAuthKey"
	licenseKeyPath = "flantIntegration.internal.licenseKey"
)

func registrationHandler(input *go_hook.HookInput, dc dependency.Container) error {
	data, err := createMadisonPayload(input.Values, dc)
	if err != nil {
		return err
	}

	_, ok := input.Values.GetOk(madisonKeyPath)
	if ok {
		return nil
	}

	licenseKey, ok := input.Values.GetOk(licenseKeyPath)
	if !ok {
		return nil
	}

	data.Type = "prometheus"

	// form request to d8-connect proxy
	req, err := newRegistrationRequest(registrationURL, data, licenseKey.String())
	if err != nil {
		input.LogEntry.Errorf("http request failed: %v", err)
		return nil
	}

	// call
	authKey, err := doMadisonRequest(req, dc, input.LogEntry)
	if err != nil {
		err := fmt.Errorf("cannot register in madison (%s %s): %v", req.Method, req.URL, err)
		input.LogEntry.Errorf(err.Error())
		return err
	}
	if authKey != "" {
		input.ConfigValues.Set(madisonKeyPath, authKey)
	}

	return nil
}

type madisonRequestData struct {
	Type          string    `json:"type,omitempty"`
	Name          string    `json:"name"`
	PrometheusURL string    `json:"prometheus_url"`
	GrafanaURL    string    `json:"grafana_url"`
	ExtraData     extraData `json:"extra_data"`
}

type extraData struct {
	Labels map[string]string `json:"labels"`
}

func createMadisonPayload(values *go_hook.PatchableValues, dc dependency.Container) (madisonRequestData, error) {
	data := madisonRequestData{}

	schema := "http"
	publicDomain := values.Get("global.modules.publicDomainTemplate").String()

	if publicDomain != "" {
		globalHTTPSMode := values.Get("global.modules.https.mode").String()

		kubeCl, err := dc.GetK8sClient()
		if err != nil {
			return madisonRequestData{}, fmt.Errorf("cannot init Kubernetes client: %v", err)
		}

		cm, err := kubeCl.CoreV1().
			ConfigMaps("d8-system").
			Get(context.TODO(), "deckhouse", metav1.GetOptions{})
		if err != nil {
			return madisonRequestData{}, fmt.Errorf("cannot get configmap deckhouse")
		}

		prometheusHTTPSMode := ""
		prometheusData, ok := cm.Data["prometheus"]
		if ok {
			var prometheus struct{ HTTPS struct{ Mode string } }
			err := yaml.Unmarshal([]byte(prometheusData), &prometheus)
			if err == nil {
				prometheusHTTPSMode = prometheus.HTTPS.Mode
			}
		}

		if prometheusHTTPSMode == "" {
			if globalHTTPSMode != "Disabled" {
				schema = "https"
			}
		} else if prometheusHTTPSMode != "Disabled" {
			schema = "https"
		}
	}

	data.GrafanaURL = schema + "://" + fmt.Sprintf(publicDomain, "grafana")
	data.PrometheusURL = data.GrafanaURL + "/prometheus"

	return data, nil
}

type madisonAuthKeyResp struct {
	Error   string `json:"error"`
	AuthKey string `json:"auth_key"`
}

// doMadisonRequest makes auth request and expect response in form of Madison API
func doMadisonRequest(req *http.Request, dc dependency.Container, logEntry *logrus.Entry) (string, error) {
	resp, err := dc.GetHTTPClient().Do(req)
	if err != nil {
		logEntry.Errorf("http call failed: %s", err)
		return "", nil
	}
	defer resp.Body.Close()

	var madisonResp madisonAuthKeyResp
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("cannot read response body: %v", err)
	}
	err = json.Unmarshal(body, &madisonResp)
	if err != nil {
		logEntry.Errorf("json unmarshaling failed, body=%q: %v", body, err)
		return "", err
	}

	if madisonResp.Error != "" {
		return "", fmt.Errorf(madisonResp.Error)
	}

	return madisonResp.AuthKey, nil
}

type registrationData struct {
	Payload string `json:"madisonData"`
}

func newRegistrationRequest(endpoint string, data madisonRequestData, key string) (*http.Request, error) {
	madisonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal madison request data")
	}
	proxyData := registrationData{
		Payload: string(madisonData),
	}
	proxyPayload, err := json.Marshal(proxyData)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal d8-connect request data")
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(proxyPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}
