package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	lokiServerAddress       = "serverAddress"
	lokiMetricName          = "metricName"
	lokiQuery               = "query"
	lokiThreshold           = "threshold"
	lokiActivationThreshold = "activationThreshold"
	lokiNamespace           = "namespace"
	lokiCortexScopeOrgID    = "cortexOrgID"
	lokiCortexHeaderKey     = "X-Scope-OrgID"
	lokiIgnoreNullValues    = "ignoreNullValues"
)

var (
	lokiDefaultIgnoreNullValues = true
)

type lokiScaler struct {
	metricType v2.MetricTargetType
	metadata   *lokiMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type lokiMetadata struct {
	serverAddress       string
	metricName          string
	query               string
	threshold           float64
	activationThreshold float64
	lokiAuth            *authentication.AuthMeta
	scalerIndex         int
	cortexOrgID         string
	ignoreNullValues    bool
	unsafeSsl           bool
}

type lokiQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func NewLokiScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	logger := InitializeLogger(config, "loki_scaler")

	meta, err := parseLokiMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing loki metadata: %s", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.unsafeSsl)

	return &lokiScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

func parseLokiMetadata(config *ScalerConfig) (meta *lokiMetadata, err error) {

	meta = &lokiMetadata{}

	if val, ok := config.TriggerMetadata[lokiServerAddress]; ok && val != "" {
		meta.serverAddress = val
	} else {
		return nil, fmt.Errorf("no %s given", lokiServerAddress)
	}

	if val, ok := config.TriggerMetadata[lokiQuery]; ok && val != "" {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no %s given", lokiQuery)
	}

	if val, ok := config.TriggerMetadata[lokiMetricName]; ok && val != "" {
		meta.metricName = val
	} else {
		return nil, fmt.Errorf("no %s given", lokiMetricName)
	}

	if val, ok := config.TriggerMetadata[lokiThreshold]; ok && val != "" {
		t, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %s", lokiThreshold, err)
		}

		meta.threshold = t
	} else {
		return nil, fmt.Errorf("no %s given", lokiThreshold)
	}

	meta.activationThreshold = 0
	if val, ok := config.TriggerMetadata[lokiActivationThreshold]; ok {
		t, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationThreshold parsing error %s", err.Error())
		}

		meta.activationThreshold = t
	}

	if val, ok := config.TriggerMetadata[lokiCortexScopeOrgID]; ok && val != "" {
		meta.cortexOrgID = val
	}

	meta.ignoreNullValues = lokiDefaultIgnoreNullValues
	if val, ok := config.TriggerMetadata[lokiIgnoreNullValues]; ok && val != "" {
		ignoreNullValues, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("err incorrect value for ignoreNullValues given: %s, "+
				"please use true or false", val)
		}
		meta.ignoreNullValues = ignoreNullValues
	}

	meta.unsafeSsl = false
	if val, ok := config.TriggerMetadata[unsafeSsl]; ok && val != "" {
		unsafeSslValue, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %s", unsafeSsl, err)
		}

		meta.unsafeSsl = unsafeSslValue
	}

	meta.scalerIndex = config.ScalerIndex

	// parse auth configs from ScalerConfig
	meta.lokiAuth, err = authentication.GetAuthConfigs(config.TriggerMetadata, config.AuthParams)
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func (s *lokiScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := s.ExecuteLokiQuery(ctx)
	if err != nil {
		s.logger.Error(err, "error executing loki query")
		return false, err
	}

	return val > s.metadata.activationThreshold, nil
}

func (s *lokiScaler) Close(context.Context) error {
	return nil
}

func (s *lokiScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("loki-%s", s.metadata.metricName))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.threshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *lokiScaler) ExecuteLokiQuery(ctx context.Context) (float64, error) {

	u, err := url.ParseRequestURI(s.metadata.serverAddress)
	if err != nil {
		return -1, err
	}
	u.Path = "/loki/api/v1/query"

	u.RawQuery = url.Values{
		"query": []string{s.metadata.query},
	}.Encode()

	req, err := http.NewRequestWithContext(context.Background(), "GET", u.String(), nil)
	if err != nil {
		return -1, err
	}

	if s.metadata.lokiAuth != nil && s.metadata.lokiAuth.EnableBasicAuth {
		req.SetBasicAuth(s.metadata.lokiAuth.Username, s.metadata.lokiAuth.Password)
	}

	if s.metadata.cortexOrgID != "" {
		req.Header.Add(lokiCortexHeaderKey, s.metadata.cortexOrgID)
	}

	r, err := s.httpClient.Do(req)
	if err != nil {
		return -1, err
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return -1, err
	}
	_ = r.Body.Close()

	if !(r.StatusCode >= 200 && r.StatusCode <= 299) {
		err := fmt.Errorf("loki query api returned error. status: %d response: %s", r.StatusCode, string(b))
		s.logger.Error(err, "loki query api returned error")
		return -1, err
	}

	var result lokiQueryResult
	err = json.Unmarshal(b, &result)
	if err != nil {
		return -1, err
	}

	var v float64 = -1

	// allow for zero element or single element result sets
	if len(result.Data.Result) == 0 {
		if s.metadata.ignoreNullValues {
			return 0, nil
		}
		return -1, fmt.Errorf("loki metrics %s target may be lost, the result is empty", s.metadata.metricName)
	} else if len(result.Data.Result) > 1 {
		return -1, fmt.Errorf("loki query %s returned multiple elements", s.metadata.query)
	}

	valueLen := len(result.Data.Result[0].Value)
	if valueLen == 0 {
		if s.metadata.ignoreNullValues {
			return 0, nil
		}
		return -1, fmt.Errorf("loki metrics %s target may be lost, the value list is empty", s.metadata.metricName)
	} else if valueLen < 2 {
		return -1, fmt.Errorf("loki query %s didn't return enough values", s.metadata.query)
	}

	val := result.Data.Result[0].Value[1]
	if val != nil {
		str := val.(string)
		v, err = strconv.ParseFloat(str, 64)
		if err != nil {
			s.logger.Error(err, "Error converting loki value", "loki_value", str)
			return -1, err
		}
	}

	return v, nil
}

func (s *lokiScaler) GetMetrics(ctx context.Context, metricName string, _ labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := s.ExecuteLokiQuery(ctx)
	if err != nil {
		s.logger.Error(err, "error executing prometheus query")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
