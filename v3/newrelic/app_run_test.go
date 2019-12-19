package newrelic

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
)

func TestResponseCodeIsError(t *testing.T) {
	cfg := defaultConfig()
	cfg.ErrorCollector.IgnoreStatusCodes = append(cfg.ErrorCollector.IgnoreStatusCodes, 504)
	run := newAppRun(cfg, internal.ConnectReplyDefaults())

	for _, tc := range []struct {
		Code    int
		IsError bool
	}{
		{Code: 0, IsError: false}, // gRPC
		{Code: 1, IsError: true},  // gRPC
		{Code: 5, IsError: false}, // gRPC
		{Code: 6, IsError: true},  // gRPC
		{Code: 99, IsError: true},
		{Code: 100, IsError: false},
		{Code: 199, IsError: false},
		{Code: 200, IsError: false},
		{Code: 300, IsError: false},
		{Code: 399, IsError: false},
		{Code: 400, IsError: true},
		{Code: 404, IsError: false},
		{Code: 503, IsError: true},
		{Code: 504, IsError: false},
	} {
		if is := run.responseCodeIsError(tc.Code); is != tc.IsError {
			t.Errorf("responseCodeIsError for %d, wanted=%v got=%v",
				tc.Code, tc.IsError, is)
		}
	}

}

func TestCrossAppTracingEnabled(t *testing.T) {
	// CAT should be enabled by default.
	cfg := defaultConfig()
	run := newAppRun(cfg, internal.ConnectReplyDefaults())
	if enabled := run.Config.CrossApplicationTracer.Enabled; !enabled {
		t.Error(enabled)
	}

	// DT gets priority over CAT.
	cfg = defaultConfig()
	cfg.DistributedTracer.Enabled = true
	cfg.CrossApplicationTracer.Enabled = true
	run = newAppRun(cfg, internal.ConnectReplyDefaults())
	if enabled := run.Config.CrossApplicationTracer.Enabled; enabled {
		t.Error(enabled)
	}

	cfg = defaultConfig()
	cfg.DistributedTracer.Enabled = false
	cfg.CrossApplicationTracer.Enabled = false
	run = newAppRun(cfg, internal.ConnectReplyDefaults())
	if enabled := run.Config.CrossApplicationTracer.Enabled; enabled {
		t.Error(enabled)
	}

	cfg = defaultConfig()
	cfg.DistributedTracer.Enabled = false
	cfg.CrossApplicationTracer.Enabled = true
	run = newAppRun(cfg, internal.ConnectReplyDefaults())
	if enabled := run.Config.CrossApplicationTracer.Enabled; !enabled {
		t.Error(enabled)
	}
}

func TestTxnTraceThreshold(t *testing.T) {
	// Test that the default txn trace threshold is the failing apdex.
	cfg := defaultConfig()
	run := newAppRun(cfg, internal.ConnectReplyDefaults())
	threshold := run.txnTraceThreshold(1 * time.Second)
	if threshold != 4*time.Second {
		t.Error(threshold)
	}

	// Test that the trace threshold can be assigned to a fixed value.
	cfg = defaultConfig()
	cfg.TransactionTracer.Threshold.IsApdexFailing = false
	cfg.TransactionTracer.Threshold.Duration = 3 * time.Second
	run = newAppRun(cfg, internal.ConnectReplyDefaults())
	threshold = run.txnTraceThreshold(1 * time.Second)
	if threshold != 3*time.Second {
		t.Error(threshold)
	}

	// Test that the trace threshold can be overwritten by server-side-config.
	// with "apdex_f".
	cfg = defaultConfig()
	cfg.TransactionTracer.Threshold.IsApdexFailing = false
	cfg.TransactionTracer.Threshold.Duration = 3 * time.Second
	reply := internal.ConnectReplyDefaults()
	json.Unmarshal([]byte(`{"agent_config":{"transaction_tracer.transaction_threshold":"apdex_f"}}`), &reply)
	run = newAppRun(cfg, reply)
	threshold = run.txnTraceThreshold(1 * time.Second)
	if threshold != 4*time.Second {
		t.Error(threshold)
	}

	// Test that the trace threshold can be overwritten by server-side-config.
	// with a numberic value.
	cfg = defaultConfig()
	reply = internal.ConnectReplyDefaults()
	json.Unmarshal([]byte(`{"agent_config":{"transaction_tracer.transaction_threshold":3}}`), &reply)
	run = newAppRun(cfg, reply)
	threshold = run.txnTraceThreshold(1 * time.Second)
	if threshold != 3*time.Second {
		t.Error(threshold)
	}
}

func TestEmptyReplyEventHarvestDefaults(t *testing.T) {
	var run internal.HarvestConfigurer = newAppRun(defaultConfig(), &internal.ConnectReply{})
	assertHarvestConfig(t, &run, expectHarvestConfig{
		maxTxnEvents:    internal.MaxTxnEvents,
		maxCustomEvents: internal.MaxCustomEvents,
		maxErrorEvents:  internal.MaxErrorEvents,
		maxSpanEvents:   internal.MaxSpanEvents,
		periods: map[internal.HarvestTypes]time.Duration{
			internal.HarvestTypesAll: 60 * time.Second,
			0:                        60 * time.Second,
		},
	})
}

func TestEventHarvestFieldsAllPopulated(t *testing.T) {
	reply, err := internal.ConstructConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 5000,
				"harvest_limits": {
					"analytic_event_data": 1,
					"custom_event_data": 2,
					"span_event_data": 3,
					"error_event_data": 4
				}
			}
		}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	var run internal.HarvestConfigurer = newAppRun(defaultConfig(), reply)
	assertHarvestConfig(t, &run, expectHarvestConfig{
		maxTxnEvents:    1,
		maxCustomEvents: 2,
		maxErrorEvents:  4,
		maxSpanEvents:   3,
		periods: map[internal.HarvestTypes]time.Duration{
			internal.HarvestMetricsTraces: 60 * time.Second,
			internal.HarvestTypesEvents:   5 * time.Second,
		},
	})
}

func TestZeroReportPeriod(t *testing.T) {
	reply, err := internal.ConstructConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 0
			}
		}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	var run internal.HarvestConfigurer = newAppRun(defaultConfig(), reply)
	assertHarvestConfig(t, &run, expectHarvestConfig{
		maxTxnEvents:    internal.MaxTxnEvents,
		maxCustomEvents: internal.MaxCustomEvents,
		maxErrorEvents:  internal.MaxErrorEvents,
		maxSpanEvents:   internal.MaxSpanEvents,
		periods: map[internal.HarvestTypes]time.Duration{
			internal.HarvestTypesAll: 60 * time.Second,
			0:                        60 * time.Second,
		},
	})
}

func TestEventHarvestFieldsOnlySpanEvents(t *testing.T) {
	reply, err := internal.ConstructConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 5000,
				"harvest_limits": { "span_event_data": 3 }
			}}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	var run internal.HarvestConfigurer = newAppRun(defaultConfig(), reply)
	assertHarvestConfig(t, &run, expectHarvestConfig{
		maxTxnEvents:    internal.MaxTxnEvents,
		maxCustomEvents: internal.MaxCustomEvents,
		maxErrorEvents:  internal.MaxErrorEvents,
		maxSpanEvents:   3,
		periods: map[internal.HarvestTypes]time.Duration{
			internal.HarvestTypesAll ^ internal.HarvestSpanEvents: 60 * time.Second,
			internal.HarvestSpanEvents:                            5 * time.Second,
		},
	})
}

func TestEventHarvestFieldsOnlyTxnEvents(t *testing.T) {
	reply, err := internal.ConstructConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 5000,
				"harvest_limits": { "analytic_event_data": 3 }
			}}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	var run internal.HarvestConfigurer = newAppRun(defaultConfig(), reply)
	assertHarvestConfig(t, &run, expectHarvestConfig{
		maxTxnEvents:    3,
		maxCustomEvents: internal.MaxCustomEvents,
		maxErrorEvents:  internal.MaxErrorEvents,
		maxSpanEvents:   internal.MaxSpanEvents,
		periods: map[internal.HarvestTypes]time.Duration{
			internal.HarvestTypesAll ^ internal.HarvestTxnEvents: 60 * time.Second,
			internal.HarvestTxnEvents:                            5 * time.Second,
		},
	})
}

func TestEventHarvestFieldsOnlyErrorEvents(t *testing.T) {
	reply, err := internal.ConstructConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 5000,
				"harvest_limits": { "error_event_data": 3 }
			}}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	var run internal.HarvestConfigurer = newAppRun(defaultConfig(), reply)
	assertHarvestConfig(t, &run, expectHarvestConfig{
		maxTxnEvents:    internal.MaxTxnEvents,
		maxCustomEvents: internal.MaxCustomEvents,
		maxErrorEvents:  3,
		maxSpanEvents:   internal.MaxSpanEvents,
		periods: map[internal.HarvestTypes]time.Duration{
			internal.HarvestTypesAll ^ internal.HarvestErrorEvents: 60 * time.Second,
			internal.HarvestErrorEvents:                            5 * time.Second,
		},
	})
}

func TestEventHarvestFieldsOnlyCustomEvents(t *testing.T) {
	reply, err := internal.ConstructConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 5000,
				"harvest_limits": { "custom_event_data": 3 }
			}}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	var run internal.HarvestConfigurer = newAppRun(defaultConfig(), reply)
	assertHarvestConfig(t, &run, expectHarvestConfig{
		maxTxnEvents:    internal.MaxTxnEvents,
		maxCustomEvents: 3,
		maxErrorEvents:  internal.MaxErrorEvents,
		maxSpanEvents:   internal.MaxSpanEvents,
		periods: map[internal.HarvestTypes]time.Duration{
			internal.HarvestTypesAll ^ internal.HarvestCustomEvents: 60 * time.Second,
			internal.HarvestCustomEvents:                            5 * time.Second,
		},
	})
}

func TestConfigurableHarvestNegativeReportPeriod(t *testing.T) {
	h, err := internal.ConstructConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": -1
			}}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	expect := time.Duration(internal.DefaultConfigurableEventHarvestMs) * time.Millisecond
	if period := h.ConfigurablePeriod(); period != expect {
		t.Fatal(expect, period)
	}
}

func TestReplyTraceIDGenerator(t *testing.T) {
	// Test that the default connect reply has a populated trace id
	// generator that works.
	reply := internal.ConnectReplyDefaults()
	id1 := reply.TraceIDGenerator.GenerateTraceID()
	id2 := reply.TraceIDGenerator.GenerateTraceID()
	if len(id1) != 32 || len(id2) != 32 || id1 == id2 {
		t.Error(id1, id2)
	}
	spanID1 := reply.TraceIDGenerator.GenerateSpanID()
	spanID2 := reply.TraceIDGenerator.GenerateSpanID()
	if len(spanID1) != 16 || len(spanID2) != 16 || spanID1 == spanID2 {
		t.Error(spanID1, spanID2)
	}
}

func TestConfigurableTxnEvents_withCollResponse(t *testing.T) {
	h, err := internal.ConstructConnectReply([]byte(
		`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 10000,
                "harvest_limits": {
             		"analytic_event_data": 15
                }
			}
        }}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	result := newAppRun(defaultConfig(), h).MaxTxnEvents()
	if result != 15 {
		t.Error(fmt.Sprintf("Unexpected max number of txn events, expected %d but got %d", 15, result))
	}
}

func TestConfigurableTxnEvents_notInCollResponse(t *testing.T) {
	reply, err := internal.ConstructConnectReply([]byte(
		`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 10000
			}
        }}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	expected := 10
	cfg := defaultConfig()
	cfg.TransactionEvents.MaxSamplesStored = expected
	result := newAppRun(cfg, reply).MaxTxnEvents()
	if result != expected {
		t.Error(fmt.Sprintf("Unexpected max number of txn events, expected %d but got %d", expected, result))
	}
}

func TestConfigurableTxnEvents_configMoreThanMax(t *testing.T) {
	h, err := internal.ConstructConnectReply([]byte(
		`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 10000
			}
        }}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	cfg := defaultConfig()
	cfg.TransactionEvents.MaxSamplesStored = internal.MaxTxnEvents + 100
	result := newAppRun(cfg, h).MaxTxnEvents()
	if result != internal.MaxTxnEvents {
		t.Error(fmt.Sprintf("Unexpected max number of txn events, expected %d but got %d", internal.MaxTxnEvents, result))
	}
}

type expectHarvestConfig struct {
	maxTxnEvents    int
	maxCustomEvents int
	maxErrorEvents  int
	maxSpanEvents   int
	periods         map[internal.HarvestTypes]time.Duration
}

func assertHarvestConfig(t testing.TB, hc *internal.HarvestConfigurer, expect expectHarvestConfig) {
	if h, ok := t.(interface {
		Helper()
	}); ok {
		h.Helper()
	}
	if max := (*hc).MaxTxnEvents(); max != expect.maxTxnEvents {
		t.Error(max, expect.maxTxnEvents)
	}
	if max := (*hc).MaxCustomEvents(); max != expect.maxCustomEvents {
		t.Error(max, expect.maxCustomEvents)
	}
	if max := (*hc).MaxSpanEvents(); max != expect.maxSpanEvents {
		t.Error(max, expect.maxSpanEvents)
	}
	if max := (*hc).MaxErrorEvents(); max != expect.maxErrorEvents {
		t.Error(max, expect.maxErrorEvents)
	}
	if periods := (*hc).ReportPeriods(); !reflect.DeepEqual(periods, expect.periods) {
		t.Error(periods, expect.periods)
	}
}
