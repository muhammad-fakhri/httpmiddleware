package httpmiddleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/muhammad-fakhri/log"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

// IngressLog represents concrete type of the middleware
type IngressLog struct {
	logger log.Logger
	config *Config
}

type IngressLogger interface {
	Enforce(next http.Handler) http.Handler
	EnforceWithParams(next httprouter.Handle) httprouter.Handle
}

// LogMessage is a struct to keep the log message easier
type LogMessage struct {
	URL            string
	ReqMethod      string
	ReqHeader      http.Header
	ReqBody        string
	ResponseHeader http.Header
	ResponseCode   int
	ResponseBody   string
	TimeTakenInMS  int64
}

const (
	valueLogTypeIngress = "ingress_http"
)

type LogRequest struct {
	URL    string
	Method string
	Header http.Header
	Body   string
}

// NewIngressLogMiddleware is to initialize ingress log middleware object
func NewIngressLogMiddleware(logger log.Logger, optionalConfig ...*Config) *IngressLog {
	var conf *Config
	if len(optionalConfig) == 0 || optionalConfig[0] == nil {
		conf = defaultConfig()
	} else {
		conf = NewConfig(optionalConfig[0])
	}

	return &IngressLog{
		logger: logger,
		config: conf,
	}
}

// Enforce is to apply log ingress middleware to the 'next' handler
func (i *IngressLog) Enforce(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logReqMessage := buildLogRequest(r)

		newRequest := i.appendContextDataAndSetValue(r, i.logger)
		newWriter := i.logger.CreateResponseWrapper(w)

		var (
			startTime       time.Time
			elapsedTimeInMS int64
		)

		defer func(ctx context.Context, request *LogRequest, elapsedTimeInMS *int64, requestTimestamp *time.Time, writer *log.LoggingResponseWriter) {
			r := recover()
			if r != nil {
				fmt.Println("[ingress][panic] recovered from: ", r)
				debug.PrintStack()

				// default panic value
				writer.WriteHeader(http.StatusInternalServerError)
				writer.Write([]byte(fmt.Sprintf("panic: %v.", r)))
			}

			i.log(newRequest.Context(), request, *elapsedTimeInMS, *requestTimestamp, writer)

		}(newRequest.Context(), logReqMessage, &elapsedTimeInMS, &startTime, newWriter)

		startTime = time.Now()
		next.ServeHTTP(newWriter, newRequest)
		elapsedTimeInMS = time.Since(startTime).Milliseconds()

	})
}

// EnforceWithParams is to apply log ingress middleware to the 'next' handler. Like http.HandlerFunc,
// but has a third parameter for the values of wildcards (variables), e.g: github.com/julienschmidt/httprouter
func (i *IngressLog) EnforceWithParams(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		logReqMessage := buildLogRequest(r)

		newRequest := i.appendContextDataAndSetValue(r, i.logger)
		newWriter := i.logger.CreateResponseWrapper(w)

		var (
			startTime       time.Time
			elapsedTimeInMS int64
		)

		defer func(ctx context.Context, reqmes *LogRequest, elapsedTimeInMS *int64, requestTimestamp *time.Time, writer *log.LoggingResponseWriter) {
			r := recover()
			if r != nil {
				fmt.Println("[ingress][panic] recovered from: ", r)
				debug.PrintStack()

				// default panic value
				writer.WriteHeader(http.StatusInternalServerError)
				writer.Write([]byte(fmt.Sprintf("panic: %v.", r)))
			}

			i.log(newRequest.Context(), reqmes, *elapsedTimeInMS, *requestTimestamp, writer)

		}(newRequest.Context(), logReqMessage, &elapsedTimeInMS, &startTime, newWriter)

		startTime = time.Now()
		next(newWriter, newRequest, ps)
		elapsedTimeInMS = time.Since(startTime).Milliseconds()

	}
}

func (i *IngressLog) log(ctx context.Context, request *LogRequest, timeTaken int64, requestTimestamp time.Time, rw *log.LoggingResponseWriter) {
	if i.config.DisableIngressLog || (i.config.LogFailedRequestOnly() && rw.Status == http.StatusOK) {
		// skip ingress log, rely on load balancer log or custom log instead
		return
	}

	// construct data map
	dataMap := make(map[string]interface{})
	dataMap[FieldType] = valueLogTypeIngress
	dataMap[FieldURL] = fmt.Sprintf("%s %s", request.Method, request.URL)
	dataMap[FieldReqTimestamp] = requestTimestamp.Unix()
	dataMap[FieldStatus] = rw.Status
	dataMap[FieldDurationMs] = timeTaken

	if i.config.LogRequestHeader() {
		header := request.Header.Clone()
		header.Del("Authorization")

		excludeRequestHeaderKeys := i.config.ExcludeOpt.RequestHeaderKeys
		if len(excludeRequestHeaderKeys) > 0 {
			for _, headerKey := range excludeRequestHeaderKeys {
				header.Del(headerKey)
			}
		}

		dataMap[FieldReqHeader] = header
	}

	if i.config.LogRequestBody() {
		dataMap[FieldReqBody] = request.Body
	}

	if i.config.LogResponseHeader() {
		header := rw.Header().Clone()
		header.Del("Authorization")
		dataMap[FieldResponseHeader] = header
	}

	if i.config.LogResponseBody() {
		if i.config.LogSuccessResponseBody() {
			dataMap[FieldResponseBody] = rw.Body
		} else {
			if rw.Status != http.StatusOK {
				dataMap[FieldResponseBody] = rw.Body
			} else {
				dataMap[FieldResponseBody] = wipedMessage
			}
		}
	}

	i.logger.InfoMap(ctx, dataMap)

}

func buildLogRequest(r *http.Request) *LogRequest {
	return &LogRequest{
		URL:    r.URL.String(),
		Method: r.Method,
		Header: r.Header,
		Body:   getRequestBody(r),
	}
}

func getRequestBody(request *http.Request) string {
	if request.Body == nil {
		return "null"
	}

	requestBodyBytes, err := getBodyBytes(&request.Body)
	if err != nil {
		return "null"
	}

	return string(requestBodyBytes)
}

func getBodyBytes(body *io.ReadCloser) ([]byte, error) {
	responseBodyBytes, err := ioutil.ReadAll(*body)
	*body = ioutil.NopCloser(bytes.NewBuffer(responseBodyBytes))
	return responseBodyBytes, err
}

func (i *IngressLog) appendContextDataAndSetValue(r *http.Request, l log.Logger) *http.Request {
	v := r.Context().Value(log.ContextDataMapKey)
	if v != nil {
		return r
	}

	var contextID string
	if contextID = r.Header.Get(headerNameRequestID); contextID == "" {
		contextID = uuid.New().String()
	}

	// TODO: add common fields to be logged in http
	return l.SetContextDataAndSetValue(r, nil, contextID)
}
