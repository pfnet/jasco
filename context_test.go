package jasco

import (
	"bytes"
	"encoding/json"
	"github.com/gocraft/web"
	"github.com/mattn/go-scan"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func jscan(js interface{}, path string) interface{} {
	var v interface{}
	if err := scan.ScanTree(js, path, &v); err != nil {
		return nil
	}
	return v
}

func TestContext(t *testing.T) {
	router := New("", nil)
	router.Get("/", func(c *Context, rw web.ResponseWriter, req *web.Request) {
		c.RenderRaw(299, map[string]interface{}{
			"reqid": c.RequestID(),
		})
	})
	router.Post("/", func(c *Context, rw web.ResponseWriter, req *web.Request) {
		// TODO: test the content of log
		c.AddLogField("action", "post /")
		var m map[string]interface{}
		if err := c.ParseBody(&m); err != nil {
			c.ErrLog(err.Err).Error("Cannot parse json")
			c.RenderError(err)
			return
		}
		c.Log().WithField("data", m).Info("body")
		c.Render(m)
	})
	s := httptest.NewServer(router)
	defer s.Close()

	parseRes := func(res *http.Response) (map[string]interface{}, error) {
		d, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, err
		}
		var m map[string]interface{}
		if err := json.Unmarshal(d, &m); err != nil {
			return nil, err
		}
		return m, nil
	}
	get := func(path string) (*http.Response, map[string]interface{}, error) {
		res, err := http.Get(s.URL + path)
		if err != nil {
			return nil, nil, err
		}
		m, err := parseRes(res)
		return res, m, err
	}
	post := func(path string, data interface{}) (*http.Response, map[string]interface{}, error) {
		j, err := json.Marshal(data)
		if err != nil {
			return nil, nil, err
		}
		res, err := http.Post(s.URL+path, "application/json", bytes.NewReader(j))
		if err != nil {
			return nil, nil, err
		}
		m, err := parseRes(res)
		return res, m, err
	}

	Convey("Given a context and a server using it", t, func() {
		Convey("when getting a content", func() {
			res, m, err := get("/")
			So(err, ShouldBeNil)
			So(res.StatusCode, ShouldEqual, 299)

			Convey("the response should contain reqid", func() {
				So(m["reqid"], ShouldBeGreaterThanOrEqualTo, 1)
			})
		})

		Convey("when posting JSON", func() {
			req := map[string]interface{}{
				"a": "b",
				"c": 1.0,
			}
			res, m, err := post("/", req)
			So(err, ShouldBeNil)
			So(res.StatusCode, ShouldEqual, http.StatusOK)

			Convey("the response should be same as the request", func() {
				So(m, ShouldResemble, req)
			})
		})

		Convey("when posting a broken JSON", func() {
			res, m, err := post("/", json.RawMessage("broken!"))
			So(err, ShouldBeNil)

			Convey("it should fail", func() {
				So(res.StatusCode, ShouldEqual, http.StatusBadRequest)
			})

			Convey("it should contain error information", func() {
				So(jscan(m, "/error/code"), ShouldEqual, requestBodyParseErrorCode)
				sid := jscan(m, "/error/request_id").(string)
				id, err := strconv.Atoi(sid)
				So(err, ShouldBeNil)
				So(id, ShouldBeGreaterThanOrEqualTo, 1)
				So(jscan(m, "/error/message"), ShouldNotBeBlank)
			})
		})

		Convey("when getting a nonexistent path", func() {
			res, m, err := get("/hoge")
			So(err, ShouldBeNil)

			Convey("the status code should be 404", func() {
				So(res.StatusCode, ShouldEqual, http.StatusNotFound)
			})

			Convey("the response should contain error information", func() {
				So(jscan(m, "/error/code"), ShouldEqual, requestURLNotFoundErrorCode)
				So(jscan(m, "/error/message"), ShouldNotBeBlank)
			})
		})
	})
}
