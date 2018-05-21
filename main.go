package main

import (
	"bufio"
	"compress/flate"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// Handler
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func main() {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(Flate())

	// Routes
	e.GET("/", hello)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}

type flateResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func Flate() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			res := c.Response()
			res.Header().Add(echo.HeaderVary, echo.HeaderAcceptEncoding)
			if strings.Contains(c.Request().Header.Get(echo.HeaderAcceptEncoding), "flate") {
				res.Header().Set(echo.HeaderContentEncoding, "flate")
				rw := res.Writer
				w, err := flate.NewWriter(rw, -1)
				if err != nil {
					return err
				}
				defer func() {
					if res.Size == 0 {
						if res.Header().Get(echo.HeaderContentEncoding) == "flate" {
							res.Header().Del(echo.HeaderContentEncoding)
						}
						res.Writer = rw
						w.Reset(ioutil.Discard)
					}
					w.Close()
				}()
				grw := &flateResponseWriter{Writer: w, ResponseWriter: rw}
				res.Writer = grw
			}
			return next(c)
		}
	}
}

func (w *flateResponseWriter) WriteHeader(code int) {
	if code == http.StatusNoContent { // Issue #489
		w.ResponseWriter.Header().Del(echo.HeaderContentEncoding)
	}
	w.Header().Del(echo.HeaderContentLength) // Issue #444
	w.ResponseWriter.WriteHeader(code)
}

func (w *flateResponseWriter) Write(b []byte) (int, error) {
	if w.Header().Get(echo.HeaderContentType) == "" {
		w.Header().Set(echo.HeaderContentType, http.DetectContentType(b))
	}
	return w.Writer.Write(b)
}

func (w *flateResponseWriter) Flush() {
	w.Writer.(*gzip.Writer).Flush()
}

func (w *flateResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

func (w *flateResponseWriter) CloseNotify() <-chan bool {
	return w.ResponseWriter.(http.CloseNotifier).CloseNotify()
}
