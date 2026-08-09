package api

import (
	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(body)
	w.bytes += n
	return n, err
}

// httpRouterHandler is the signature for functions that accepts a reqcontext.RequestContext in addition to those
// required by the httprouter package.
type httpRouterHandler func(http.ResponseWriter, *http.Request, httprouter.Params, reqcontext.RequestContext)

// wrap parses the request and adds a reqcontext.RequestContext instance related to the request.
func (rt *_router) wrap(fn httpRouterHandler) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		startedAt := time.Now()
		rt.metricsState.begin()
		tracked := &statusWriter{ResponseWriter: w}
		w = tracked
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestPayload)
		reqUUID, err := uuid.NewV4()
		if err != nil {
			rt.baseLogger.WithError(err).Error("can't generate a request UUID")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var ctx = reqcontext.RequestContext{
			ReqUUID: reqUUID,
		}

		// Create a request-specific logger
		ctx.Logger = rt.baseLogger.WithFields(logrus.Fields{
			"reqid":     ctx.ReqUUID.String(),
			"remote-ip": r.RemoteAddr,
		})
		defer func() {
			if recovered := recover(); recovered != nil {
				ctx.Logger.WithField("panic", recovered).Error("panic while handling request")
				if tracked.status == 0 {
					writeError(tracked, http.StatusInternalServerError, "Internal server error")
				}
			}
			if tracked.status == 0 {
				tracked.status = http.StatusOK
			}
			rt.metricsState.observe(tracked.status)
			ctx.Logger.WithFields(logrus.Fields{
				"method": r.Method, "path": r.URL.Path, "status": tracked.status,
				"bytes": tracked.bytes, "duration_ms": time.Since(startedAt).Milliseconds(),
			}).Info("request completed")
		}()

		// Call the next handler in chain (usually, the handler function for the path)
		fn(w, r, ps, ctx)
	}
}
