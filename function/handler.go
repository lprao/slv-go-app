package function

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/lprao/slv-go-lib/pkg/slvlib"
	slvpb "github.com/lprao/slv-proto"
	handler "github.com/openfaas/templates-sdk/go-http"
)

const (
	SensorName = "Sensor#93e00902"
)

type SensorValue struct {
	MoistureLevel int `json: "moistureLevel"`
}

func StoreSoilSensorValue(req handler.Request) (handler.Response, error) {
	var err error
	var sensorValue SensorValue
	var response handler.Response
	var slv *slvlib.SlvInt

	if err := json.Unmarshal(req.Body, &sensorValue); err != nil {
		response.StatusCode = http.StatusBadRequest
		return response, err
	}

	slvVarName := readSecret(SensorName)
	if slvVarName == "" {
		response.StatusCode = http.StatusInternalServerError
		return response, fmt.Errorf("failed to fetch SLV variable name from secret store")
	}

	slv, err = slvlib.GetSlvIntByName(slvVarName)
	if err != nil {
		slv, err = slvlib.NewSlvInt(slvVarName, 0, slvpb.VarScope_PRIVATE, slvpb.VarPermissions_READWRITE)
		if err != nil {
			response.StatusCode = http.StatusInternalServerError
			return response, err
		}
	}

	_, err = slv.Set(sensorValue.MoistureLevel)
	if err != nil {
		response.StatusCode = http.StatusInternalServerError
		return response, err
	}

	message := fmt.Sprintf("Body: %s", string("SLV updated."))

	return handler.Response{
		Body:       []byte(message),
		StatusCode: http.StatusOK,
	}, err
}

func readSecret(name string) string {
	res, err := ioutil.ReadFile(path.Join("/var/openfaas/secrets/", name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(res))
}
