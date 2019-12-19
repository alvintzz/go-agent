package internal

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

var (
	samplePayload = Payload{
		payloadCaller: payloadCaller{
			Type:    CallerTypeApp,
			Account: "123",
			App:     "456",
		},
		ID:                   "myid",
		TracedID:             "mytrip",
		Priority:             0.12345,
		Timestamp:            timestampMillis(time.Now()),
		HasNewRelicTraceInfo: true,
	}
)

func TestPayloadNil(t *testing.T) {
	out, err := AcceptPayload(nil, "123")
	if err != nil || out != nil {
		t.Fatal(err, out)
	}
}

func TestPayloadText(t *testing.T) {
	hdrs := http.Header{}
	hdrs.Set(DistributedTraceNewRelicHeader, samplePayload.NRText())
	out, err := AcceptPayload(hdrs, "123")
	if err != nil || out == nil {
		t.Fatal(err, out)
	}
	out.Timestamp = samplePayload.Timestamp // account for timezone differences
	if samplePayload != *out {
		t.Fatal(samplePayload, out)
	}
}

func TestPayloadHTTPSafe(t *testing.T) {
	hdrs := http.Header{}
	hdrs.Set(DistributedTraceNewRelicHeader, samplePayload.NRHTTPSafe())
	out, err := AcceptPayload(hdrs, "123")
	if err != nil || nil == out {
		t.Fatal(err, out)
	}
	out.Timestamp = samplePayload.Timestamp // account for timezone differences
	if samplePayload != *out {
		t.Fatal(samplePayload, out)
	}
}

func TestTimestampMillisMarshalUnmarshal(t *testing.T) {
	var sec int64 = 111
	var millis int64 = 222
	var micros int64 = 333
	var nsecWithMicros = 1000*1000*millis + 1000*micros
	var nsecWithoutMicros = 1000 * 1000 * millis

	input := time.Unix(sec, nsecWithMicros)
	expectOutput := time.Unix(sec, nsecWithoutMicros)

	var tm timestampMillis
	tm.Set(input)
	js, err := json.Marshal(tm)
	if nil != err {
		t.Fatal(err)
	}
	var out timestampMillis
	err = json.Unmarshal(js, &out)
	if nil != err {
		t.Fatal(err)
	}
	if out.Time() != expectOutput {
		t.Fatal(out.Time(), expectOutput)
	}
}

func BenchmarkPayloadText(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		samplePayload.NRText()
	}
}

func TestEmptyPayloadData(t *testing.T) {
	// does an empty payload json blob result in an invalid payload
	var payload Payload
	fixture := []byte(`{}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err == nil {
		t.Log("Expected error from empty payload data")
		t.Fail()
	}
}

func TestRequiredFieldsPayloadData(t *testing.T) {
	var payload Payload
	fixture := []byte(`{
		"ty":"App",
		"ac":"123",
		"ap":"456",
		"id":"id",
		"tr":"traceID",
		"ti":1488325987402
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err != nil {
		t.Log("Expected valid payload if ty, ac, ap, id, tr, and ti are set")
		t.Error(err)
	}
}

func TestRequiredFieldsMissingType(t *testing.T) {
	var payload Payload
	fixture := []byte(`{
		"ac":"123",
		"ap":"456",
		"id":"id",
		"tr":"traceID",
		"ti":1488325987402
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err == nil {
		t.Log("Expected error from missing Type (ty)")
		t.Fail()
	}
}

func TestRequiredFieldsMissingAccount(t *testing.T) {
	var payload Payload
	fixture := []byte(`{
		"ty":"App",
		"ap":"456",
		"id":"id",
		"tr":"traceID",
		"ti":1488325987402
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err == nil {
		t.Log("Expected error from missing Account (ac)")
		t.Fail()
	}
}

func TestRequiredFieldsMissingApp(t *testing.T) {
	var payload Payload
	fixture := []byte(`{
		"ty":"App",
		"ac":"123",
		"id":"id",
		"tr":"traceID",
		"ti":1488325987402
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err == nil {
		t.Log("Expected error from missing App (ap)")
		t.Fail()
	}
}

func TestRequiredFieldsMissingTimestamp(t *testing.T) {
	var payload Payload
	fixture := []byte(`{
		"ty":"App",
		"ac":"123",
		"ap":"456",
		"tr":"traceID"
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err == nil {
		t.Log("Expected error from missing Timestamp (ti)")
		t.Fail()
	}
}

func TestRequiredFieldsZeroTimestamp(t *testing.T) {
	var payload Payload
	fixture := []byte(`{
		"ty":"App",
		"ac":"123",
		"ap":"456",
		"tr":"traceID",
		"ti":0
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err == nil {
		t.Log("Expected error from missing Timestamp (ti)")
		t.Fail()
	}
}

func TestPayload_W3CTraceState(t *testing.T) {
	var payload Payload
	fixture := []byte(`{
		"ty":"App",
		"ac":"123",
		"ap":"456",
		"tr":"traceID",
		"ti":0,
		"id":"1234567890123456",
		"tx":"6543210987654321",
		"pr":0.24689,
        "tk":"123"
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}
	cases := map[string]string{
		"1349956@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.24689-1569367663277,rojo=00f067aa0ba902b7,congo=t61rcWkgMzE": "123@nr=0-0-123-456-1234567890123456-6543210987654321-0-0.24689-0,rojo=00f067aa0ba902b7,congo=t61rcWkgMzE",
		"rojo=00f067aa0ba902b7,1349956@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.24689-1569367663277,congo=t61rcWkgMzE": "123@nr=0-0-123-456-1234567890123456-6543210987654321-0-0.24689-0,rojo=00f067aa0ba902b7,congo=t61rcWkgMzE",
		"rojo=00f067aa0ba902b7,congo=t61rcWkgMzE,1349956@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.24689-1569367663277": "123@nr=0-0-123-456-1234567890123456-6543210987654321-0-0.24689-0,rojo=00f067aa0ba902b7,congo=t61rcWkgMzE",
		"1349956@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.24689-1569367663277":                                         "123@nr=0-0-123-456-1234567890123456-6543210987654321-0-0.24689-0",
		"": "123@nr=0-0-123-456-1234567890123456-6543210987654321-0-0.24689-0",
	}
	for k, v := range cases {
		payload.OriginalTraceState = k
		if payload.W3CTraceState() != v {
			t.Errorf("Unexpected trace state - expected %s but got %s", v, payload.W3CTraceState())
		}
	}
}

func TestProcessTraceParent(t *testing.T) {
	var payload Payload
	traceParent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	err := processTraceParent(traceParent, &payload)
	if nil != err {
		t.Errorf("Unexpected error for trace parent %s: %v", traceParent, err)
	}
	traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	if payload.TracedID != traceID {
		t.Errorf("Unexpected Trace ID in trace parent - expected %s, got %v", traceID, payload.TracedID)
	}
	spanID := "00f067aa0ba902b7"
	if payload.ID != spanID {
		t.Errorf("Unexpected Span ID in trace parent - expected %s, got %v", spanID, payload.ID)
	}
	if payload.Sampled != nil {
		t.Errorf("Expected traceparent %s sampled to be unset, but it is not", traceParent)
	}
}

func TestProcessTraceParentInvalidFormat(t *testing.T) {
	cases := []string{
		"000-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"0X-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"0-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d-00f067aa0ba902b7-01",
		"0-4bf92f3577b34da6a3ce929d0e0e47366666666-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4MMM-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b711111-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba9TTT7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-0T",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-0",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-031",
	}
	var payload Payload
	for _, traceParent := range cases {
		err := processTraceParent(traceParent, &payload)
		if nil == err {
			t.Errorf("No error reported for trace parent %s", traceParent)
		}
	}
}

func TestProcessTraceState(t *testing.T) {
	var payload Payload
	processTraceState("190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,rojo=00f067aa0ba902b7", "190", &payload)
	if payload.TrustedAccountKey != "190" {
		t.Errorf("Wrong trusted account key: expected 190 but got %s", payload.TrustedAccountKey)
	}
	if payload.Type != "Mobile" {
		t.Errorf("Wrong payload type: expected Mobile but got %s", payload.Type)
	}
	if payload.Account != "332029" {
		t.Errorf("Wrong account: expected 332029 but got %s", payload.Account)
	}
	if payload.App != "2827902" {
		t.Errorf("Wrong app ID: expected 2827902 but got %s", payload.App)
	}
	if payload.TrustedParentID != "5f474d64b9cc9b2a" {
		t.Errorf("Wrong Trusted Parent ID: expected 5f474d64b9cc9b2a but got %s", payload.ID)
	}
	if payload.TransactionID != "7d3efb1b173fecfa" {
		t.Errorf("Wrong transaction ID: expected 7d3efb1b173fecfa but got %s", payload.TransactionID)
	}
	if nil != payload.Sampled {
		t.Errorf("Payload sampled field was set when it should not be")
	}
	if payload.Priority != 0.0 {
		t.Errorf("Wrong priority: expected 0.0 but got %f", payload.Priority)
	}
	if payload.Timestamp != timestampMillis(timeFromUnixMilliseconds(1518469636035)) {
		t.Errorf("Wrong timestamp: expected 1518469636035 but got %v", payload.Timestamp)
	}
}

func TestExtractNRTraceStateEntry(t *testing.T) {
	trustedAccountID := "12345"
	cases := map[string]string{
		"12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,rojo=00f067aa0ba902b7,congo=t61rcWkgMzE":                                                                             "12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,",
		"congo=t61rcWkgMzE,12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,rojo=00f067aa0ba902b7":                                                                             "12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,",
		"12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035":                                         "12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,",
		"rojo=00f067aa0ba902b7,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,congo=t61rcWkgMzE,12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277": "12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277",
		"rojo=00f067aa0ba902b7,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,congo=t61rcWkgMzE":                                                                                          "",
		"rojo=00f067aa0ba902b7": "",
	}

	for test, expected := range cases {
		result := findTrustedNREntry(test, trustedAccountID)
		if result != expected {
			t.Errorf("Expected %s but got %s", expected, result)
		}
	}
}

func TestTracingVendors(t *testing.T) {
	thisAccount := "12345"
	cases := map[string]string{
		"12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,rojo=00f067aa0ba902b7,congo=t61rcWkgMzE":                                                                                 "rojo,congo",
		"congo=t61rcWkgMzE,12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,rojo=00f067aa0ba902b7":                                                                                 "congo,rojo",
		"12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035":                                             "190@nr",
		"atd@rojo=00f067aa0ba902b7,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,congo=t61rcWkgMzE,12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277": "atd@rojo,190@nr,congo",
		"rojo=00f067aa0ba902b7,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,fff@congo=t61rcWkgMzE":                                                                                          "rojo,190@nr,fff@congo",
		"rojo=00f067aa0ba902b7": "rojo",
		"":                      "",
	}

	for test, expected := range cases {
		p := Payload{}
		p.OriginalTraceState = test
		result := tracingVendors(&p, thisAccount)
		if result != expected {
			t.Errorf("Expected %s but got %s for case %s", expected, result, test)
		}
	}
}
