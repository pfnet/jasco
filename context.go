package jasco

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/gocraft/web"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"
)

// Context is a context object for gocraft/web.
type Context struct {
	body      []byte
	bodyError *Error

	requestID uint64

	response   web.ResponseWriter
	request    *web.Request
	httpStatus int

	logger    *logrus.Logger
	logFields logrus.Fields
}

// New creates a new web.Router based on Context.
func New(prefix string, logger *logrus.Logger) *web.Router {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	root := web.NewWithPrefix(Context{}, prefix)
	root.NotFound((*Context).NotFoundHandler)
	root.Middleware(func(c *Context, rw web.ResponseWriter, req *web.Request,
		next web.NextMiddlewareFunc) {
		c.SetLogger(logger)
		next(rw, req)
	})
	root.Middleware((*Context).setUpContext)
	return root
}

// SetLogger sets the logger to the context. The logger must be set before any
// action is invoked.
func (c *Context) SetLogger(l *logrus.Logger) {
	c.logger = l
}

// AddLogField adds a field shown in all log entries written via this context.
func (c *Context) AddLogField(key string, value interface{}) {
	c.logFields[key] = value
}

// RemoveLogField removes a field to be logged.
func (c *Context) RemoveLogField(key string) {
	delete(c.logFields, key)
}

// SetHTTPStatus sets HTTP status of the response. This method is used when an
// action doesn't render JSON.
func (c *Context) SetHTTPStatus(s int) {
	c.httpStatus = s
}

// RequestID returns the ID that is unique to the request.
func (c *Context) RequestID() uint64 {
	return c.requestID
}

var requestIDCounter uint64

func (c *Context) setUpContext(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	c.requestID = atomic.AddUint64(&requestIDCounter, 1)
	c.response = rw
	c.request = req
	c.logFields = logrus.Fields{
		"reqid": c.requestID,
	}

	start := time.Now()
	defer func() {
		elapsed := time.Now().Sub(start)
		// Use custom logging because file and line aren't necessary here.
		c.logger.WithFields(logrus.Fields{
			"reqid":   c.requestID,
			"reqtime": fmt.Sprintf("%d.%03d", int(elapsed/time.Second), int(elapsed%time.Second/time.Millisecond)),
			"method":  req.Method,
			"uri":     req.URL.RequestURI(),
			"status":  c.httpStatus,
		}).Info("Access")
	}()

	// TODO: When the process stops due to segmentation fault or OOM killer,
	// request information will be lost. To provide as much information
	// at the point of failure as possible, we need to dump stacktraces of
	// goroutines when the process fails.

	// TODO: When a request is dead locked, access information won't be logged.
	// Although it isn't possible to prevent dead locks perfectly, we can
	// implement error reporting mechanism by having a global map containing
	// all active requests. If we have a goroutine which periodically monitors
	// the map, we can report information about requests which is spending too
	// much time.
	next(rw, req)
}

// NotFoundHandler handles 404.
func (c *Context) NotFoundHandler(rw web.ResponseWriter, req *web.Request) {
	c.RenderError(NewError(requestURLNotFoundErrorCode, "The request URL was not found.",
		http.StatusNotFound, nil))
}

// Log returns the logger having meta information.
func (c *Context) Log() *logrus.Entry {
	return c.CLog(1)
}

// ErrLog returns the logger with error information.
func (c *Context) ErrLog(err error) *logrus.Entry {
	return c.CLog(1).WithField("err", err)
}

// CLog returns the logger having meta information and the information of the
// caller with the given callDepth. This method is useful when a utility
// function wants to write a log but it want to have "file" and "line" of the
// caller. When callerDepth is 0, "file" and "line" of the caller of this
// method are used.
func (c *Context) CLog(callerDepth int) *logrus.Entry {
	// TODO: This is a temporary solution until logrus support filename and line number
	_, file, line, ok := runtime.Caller(callerDepth + 1)
	if !ok {
		return c.logger.WithField("reqid", c.requestID)
	}
	file = filepath.Base(file) // only the filename at the moment
	return c.logger.WithFields(logrus.Fields{
		"file": file,
		"line": line,
	}).WithFields(c.logFields)
}

// PathParams returns parameters embedded in the URL.
func (c *Context) PathParams() *PathParams {
	return &PathParams{c.request}
}

func (c *Context) render(status int, v interface{}) {
	c.httpStatus = status

	data, err := json.Marshal(v)
	if err != nil {
		c.ErrLog(err).Error("Cannot marshal json")
		c.httpStatus = http.StatusInternalServerError
		c.response.Header().Set("Content-Type", "application/json")
		c.response.WriteHeader(c.httpStatus)
		c.response.Write([]byte(`{"error":{"code":"J0002","message":"Internal server error."}}`))
		return
	}

	c.response.Header().Set("Content-Type", "application/json")
	c.response.WriteHeader(status)
	_, err = c.response.Write(data)
	if err != nil {
		c.ErrLog(err).Error("Cannot write a response")
	}
}

// Render renders a successful result as a JSON.
func (c *Context) Render(v interface{}) {
	c.render(http.StatusOK, v)
}

// RenderRaw renders a JSON with the given status code.
func (c *Context) RenderRaw(status int, v interface{}) {
	c.render(status, v)
}

// RenderError renders a failing result as a JSON.
func (c *Context) RenderError(e *Error) {
	e.SetRequestID(c.requestID)
	c.render(e.Status, map[string]interface{}{
		"error": e,
	})
}

// Body returns a slice containing whole request body. It caches the result so
// that controllers can call this method as many time as they want.
//
// When the request body is empty (i.e. Read(req.Body) returns io.EOF), this
// method returns and caches an empty body slice (could be nil) and a nil error.
//
// Noet that this method returns an error as *Error, not error. So, the return
// value shouldn't be assigned to error type to avoid confusing nil comparison
// problem.
func (c *Context) Body() ([]byte, *Error) {
	if c.body != nil || c.bodyError != nil {
		return c.body, c.bodyError
	}

	body, err := ioutil.ReadAll(c.request.Body)
	if err != nil {
		if err != io.EOF {
			c.bodyError = NewError(requestBodyParseErrorCode,
				"Cannot read the request body", http.StatusBadRequest, err)
		}
	}

	// Close and replace with new ReadCloser for parsing
	// mime/multipart request body by Request.FormFile method.
	c.request.Body.Close()
	c.request.Body = ioutil.NopCloser(bytes.NewReader(body))
	c.body = body
	return c.body, c.bodyError
}

// ParseBody parses the request body as a JSON. The argument v is directly
// passed to json.Unmarshal.
func (c *Context) ParseBody(v interface{}) *Error {
	body, err := c.Body()
	if err != nil {
		return err
	}
	// TODO: check Content-Type
	if err := json.Unmarshal(body, v); err != nil {
		return NewError(requestBodyParseErrorCode,
			"Cannot parse the request body as JSON.",
			http.StatusBadRequest, err)
	}
	return nil
}
