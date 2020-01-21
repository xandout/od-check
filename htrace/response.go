package htrace

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"fmt"
	"strconv"
	"time"
	"crypto/tls"
	"net/http/httptrace"
	"net/textproto"
)

func NewClientTrace(url string, method string) (ClientTraceResponse, error){

	ctr := ClientTraceResponse{}
	ctr.RequestURL = url
	ctr.RequestMethod = method
	c := http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return ctr, err
	}

	trace := &httptrace.ClientTrace{

		// GetConn is called before a connection is created or
		// retrieved from an idle pool. The hostPort is the
		// "host:port" of the target or proxy. GetConn is called even
		// if there's already an idle cached connection available.
		GetConn: func(hostPort string) {
			ctr.GetConn = hostPort
		},

		// GotConn is called after a successful connection is
		// obtained. There is no hook for failure to obtain a
		// connection; instead, use the error from
		// Transport.RoundTrip.
		GotConn: func(connInfo httptrace.GotConnInfo) {
			ctr.GotConn = connInfo
		},

		// PutIdleConn is called when the connection is returned to
		// the idle pool. If err is nil, the connection was
		// successfully returned to the idle pool. If err is non-nil,
		// it describes why not. PutIdleConn is not called if
		// connection reuse is disabled via Transport.DisableKeepAlives.
		// PutIdleConn is called before the caller's Response.Body.Close
		// call returns.
		// For HTTP/2, this hook is not currently used.
		PutIdleConn: func(err error) {
			ctr.PutIdleConn = err

		},

		// GotFirstResponseByte is called when the first byte of the response
		// headers is available.
		GotFirstResponseByte: func() {
			ctr.GotFirstResponseByte = time.Now().UnixNano()

		},

		// Got100Continue is called if the server replies with a "100
		// Continue" response.
		Got100Continue: func() {
			ctr.Got100Continue = time.Now().UnixNano()

		},

		// Got1xxResponse is called for each 1xx informational response header
		// returned before the final non-1xx response. Got1xxResponse is called
		// for "100 Continue" responses, even if Got100Continue is also defined.
		// If it returns an error, the client request is aborted with that error value.
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			ctr.Got1xxResponse = append(ctr.Got1xxResponse, struct {
				Time    int64
				Code    int
				Headers textproto.MIMEHeader
			}{Time: time.Now().UnixNano(), Code: code, Headers: header})

			return nil
		},

		// DNSStart is called when a DNS lookup begins.
		DNSStart: func(dnsInfo httptrace.DNSStartInfo) {
			ctr.DNSStart = append(ctr.DNSStart, struct {
				Time         int64
				DNSStartInfo httptrace.DNSStartInfo
			}{Time: time.Now().UnixNano(), DNSStartInfo: dnsInfo})
		},

		// DNSDone is called when a DNS lookup ends.
		DNSDone: func(dnsZoneInfo httptrace.DNSDoneInfo) {
			ctr.DNSDone = append(ctr.DNSDone, struct {
				Time        int64
				DNSDoneInfo httptrace.DNSDoneInfo
			}{Time: time.Now().UnixNano(), DNSDoneInfo: dnsZoneInfo})
		},

		// ConnectStart is called when a new connection's Dial begins.
		// If net.Dialer.DualStack (IPv6 "Happy Eyeballs") support is
		// enabled, this may be called multiple times.
		ConnectStart: func(network, addr string) {
			ctr.ConnectStart = append(ctr.ConnectStart, struct {
				Time    int64
				Network string
				Address string
			}{Time: time.Now().UnixNano(), Network: network, Address: addr})

		},

		// ConnectDone is called when a new connection's Dial
		// completes. The provided err indicates whether the
		// connection completedly successfully.
		// If net.Dialer.DualStack ("Happy Eyeballs") support is
		// enabled, this may be called multiple times.
		ConnectDone: func(network, addr string, err error) {
			ctr.ConnectDone = append(ctr.ConnectDone, struct {
				Time    int64
				Network string
				Address string
				Error   error
			}{Time: time.Now().UnixNano(), Network: network, Address: addr, Error: err})

		},

		// TLSHandshakeStart is called when the TLS handshake is started. When
		// connecting to a HTTPS site via a HTTP proxy, the handshake happens after
		// the CONNECT request is processed by the proxy.
		TLSHandshakeStart: func() {
			ctr.TLSHandshakeStart = time.Now().UnixNano()

		},

		// TLSHandshakeDone is called after the TLS handshake with either the
		// successful handshake's connection state, or a non-nil error on handshake
		// failure.
		TLSHandshakeDone: func(tstate tls.ConnectionState, err error) {
			ctr.TLSHandshakeDone = struct {
				Time            int64
				ConnectionState tls.ConnectionState
				Error           error
			}{Time: time.Now().UnixNano(), ConnectionState: tstate, Error: err}

		},

		// WroteHeaderField is called after the Transport has written
		// each request header. At the time of this call the values
		// might be buffered and not yet written to the network.
		WroteHeaderField: func(key string, value []string) {
			hdr := make(map[string][]string)
			hdr[key] = value
			ctr.WroteHeaderField = append(ctr.WroteHeaderField, struct {
				Time   int64
				Header map[string][]string
			}{Time: time.Now().UnixNano(), Header: hdr})

		},

		// WroteHeaders is called after the Transport has written
		// all request headers.
		WroteHeaders: func() {
			ctr.WroteHeaders = time.Now().UnixNano()
		},

		// Wait100Continue is called if the Request specified
		// "Expected: 100-continue" and the Transport has written the
		// request headers but is waiting for "100 Continue" from the
		// server before writing the request body.
		Wait100Continue: func() {
			ctr.Wait100Continue = append(ctr.Wait100Continue, time.Now().UnixNano())

		},

		// WroteRequest is called with the result of writing the
		// request and any body. It may be called multiple times
		// in the case of retried requests.
		WroteRequest: func(wr httptrace.WroteRequestInfo) {
			ctr.WroteRequest = append(ctr.WroteRequest, struct {
				Time             int64
				WroteRequestInfo httptrace.WroteRequestInfo
			}{Time: time.Now().UnixNano(), WroteRequestInfo: wr})

		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	
	ctr.StartTime = time.Now().UnixNano()
	resp, err := c.Do(req)
	if err != nil {
		return ctr, err
	}
	ctr.FinishTime = time.Now().UnixNano()
	ctr.ResponseCode = resp.StatusCode
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	ctr.ContentLength = int64(len(bodyBytes))
	ctr.ResponseBody = bodyBytes
	if err != nil {
		return ctr, err
	}
	return ctr, nil
}
// ClientTraceResponse is a struct that holds the response to http trace callbacks. /usr/lib/golang/src/net/http/httptrace/trace.go
type ClientTraceResponse struct {
	// GetConn is called before a connection is created or
	// retrieved from an idle pool. The hostPort is the
	// "host:port" of the target or proxy. GetConn is called even
	// if there's already an idle cached connection available.
	GetConn string

	// GotConn is called after a successful connection is
	// obtained. There is no hook for failure to obtain a
	// connection; instead, use the error from
	// Transport.RoundTrip.
	GotConn httptrace.GotConnInfo

	// PutIdleConn is called when the connection is returned to
	// the idle pool. If err is nil, the connection was
	// successfully returned to the idle pool. If err is non-nil,
	// it describes why not. PutIdleConn is not called if
	// connection reuse is disabled via Transport.DisableKeepAlives.
	// PutIdleConn is called before the caller's Response.Body.Close
	// call returns.
	// For HTTP/2, this hook is not currently used.
	PutIdleConn error

	// GotFirstResponseByte is called when the first byte of the response
	// headers is available.
	GotFirstResponseByte int64

	// Got100Continue is called if the server replies with a "100
	// Continue" response.
	Got100Continue int64

	// Got1xxResponse is called for each 1xx informational response header
	// returned before the final non-1xx response. Got1xxResponse is called
	// for "100 Continue" responses, even if Got100Continue is also defined.
	// If it returns an error, the client request is aborted with that error value.
	Got1xxResponse []struct {
		Time    int64
		Code    int
		Headers textproto.MIMEHeader
	}

	// DNSStart is called when a DNS lookup begins.
	DNSStart []struct {
		Time         int64
		DNSStartInfo httptrace.DNSStartInfo
	}

	// DNSDone is called when a DNS lookup ends.
	DNSDone []struct {
		Time        int64
		DNSDoneInfo httptrace.DNSDoneInfo
	}

	// ConnectStart is called when a new connection's Dial begins.
	// If net.Dialer.DualStack (IPv6 "Happy Eyeballs") support is
	// enabled, this may be called multiple times.
	ConnectStart []struct {
		Time    int64
		Network string
		Address string
	}

	// ConnectDone is called when a new connection's Dial
	// completes. The provided err indicates whether the
	// connection completedly successfully.
	// If net.Dialer.DualStack ("Happy Eyeballs") support is
	// enabled, this may be called multiple times.
	ConnectDone []struct {
		Time    int64
		Network string
		Address string
		Error   error
	}

	// TLSHandshakeStart is called when the TLS handshake is started. When
	// connecting to a HTTPS site via a HTTP proxy, the handshake happens after
	// the CONNECT request is processed by the proxy.
	TLSHandshakeStart int64

	// TLSHandshakeDone is called after the TLS handshake with either the
	// successful handshake's connection state, or a non-nil error on handshake
	// failure.
	TLSHandshakeDone struct {
		Time            int64
		ConnectionState tls.ConnectionState
		Error           error
	}

	// WroteHeaderField is called after the Transport has written
	// each request header. At the time of this call the values
	// might be buffered and not yet written to the network.
	WroteHeaderField []struct {
		Time   int64
		Header map[string][]string
	}

	// WroteHeaders is called after the Transport has written
	// all request headers.
	WroteHeaders int64

	// Wait100Continue is called if the Request specified
	// "Expected: 100-continue" and the Transport has written the
	// request headers but is waiting for "100 Continue" from the
	// server before writing the request body.
	Wait100Continue []int64

	// WroteRequest is called with the result of writing the
	// request and any body. It may be called multiple times
	// in the case of retried requests.
	WroteRequest []struct {
		Time             int64
		WroteRequestInfo httptrace.WroteRequestInfo
	}

	// StartTime must be set by caller.  time.Now().UnixNano()
	StartTime int64

	// FinishTime must be set by caller.  time.Now().UnixNano()
	FinishTime int64

	// ResponseCode is the response body
	ResponseCode int

	// ResponseCode is the response body
	ContentLength int64

	// ResponseBytes is the response body bytes
	ResponseBody []byte

	// RequestURL is the requested URL
	RequestURL string

	// RequestMethod is the request method
	RequestMethod string
}


func (ctr *ClientTraceResponse) Influx() string {

	dur := time.Unix(0,ctr.FinishTime).Sub(time.Unix(0,ctr.StartTime)).Nanoseconds()
	influxLineFormat := "%s,%s %s %v"
	tags := make(map[string]string)
	tags["method"] = ctr.RequestMethod
	tags["server"] = ctr.RequestURL
	tags["status_code"] = strconv.Itoa(ctr.ResponseCode)
	tagString := ""
	for k,v := range tags {
		tagString += fmt.Sprintf("%v=%v,", k,v)
	}
	tagString = strings.TrimSuffix(tagString, ",")

	fields := make(map[string]string)
	fields["content_length"] = strconv.Itoa(int(ctr.ContentLength))
	fields["http_response_code"] = strconv.Itoa(int(ctr.ResponseCode))
	fields["response_time"] = strconv.Itoa(int(dur))
	fieldString := ""
	for k,v := range fields {
		fieldString += fmt.Sprintf("%v=%v,", k,v)
	}
	fieldString = strings.TrimSuffix(fieldString, ",")
	return fmt.Sprintf(influxLineFormat, "http_check", tagString, fieldString, time.Now().UnixNano())

}

func (ctr *ClientTraceResponse) RawJSON() (jsonval string, err error) {
	b, err := json.MarshalIndent(ctr, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}