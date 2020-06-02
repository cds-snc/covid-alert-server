package server

import (
	"context"
	"net"
	"net/http"

	"github.com/Shopify/goose/genmain"
	"github.com/Shopify/goose/logger"

	// "github.com/Shopify/goose/profiler"
	"github.com/Shopify/goose/safely"
	"github.com/Shopify/goose/srvutil"
	"google.golang.org/protobuf/proto"
	"gopkg.in/tomb.v2"
)

var log = logger.New("server")

type Server interface {
	genmain.Component
	Addr() *net.TCPAddr
}

func New(bind string, servlets []srvutil.Servlet) Server {
	sl := srvutil.CombineServlets(servlets...)

	sl = srvutil.UseServlet(sl,
		srvutil.RequestContextMiddleware,
		srvutil.RequestMetricsMiddleware,
		safely.Middleware,
	)

	return srvutil.NewServer(&tomb.Tomb{}, bind, sl)
}

func requestError(
	ctx context.Context, w http.ResponseWriter, err error,
	logMessage string, code int, resp proto.Message,
) result {
	if code == http.StatusInternalServerError {
		log(ctx, err).Error(logMessage)
	} else {
		log(ctx, err).Warn(logMessage)
	}

	data, err := proto.Marshal(resp)
	if err != nil {
		log(ctx, err).Error("error marshalling error response")
		return result{}
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		log(ctx, err).Warn("error writing error response")
		return result{}
	}
	return result{}
}

// returning this from s.fail and the s.retrieve makes it harder to call s.fail but forget to return.
type result struct{}
