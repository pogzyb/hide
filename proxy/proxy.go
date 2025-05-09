package proxy

import (
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

func Run(port string) {
	forwarder := func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Method()) == fasthttp.MethodConnect {
			dstConn, err := fasthttp.DialDualStackTimeout(string(ctx.Host()), time.Second*10)
			if err != nil {
				log.Info().Msgf("dial error: %v", err.Error())
				ctx.Error(err.Error(), fasthttp.StatusServiceUnavailable)
				return
			}
			defer dstConn.Close()
			_, err = ctx.Conn().Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
			if err != nil {
				log.Info().Msgf("write error: %v", err.Error())
				ctx.Error(err.Error(), fasthttp.StatusServiceUnavailable)
				return
			}
			go io.Copy(ctx.Conn(), dstConn)
			io.Copy(dstConn, ctx.Conn())
		} else {
			req := new(fasthttp.Request)
			resp := new(fasthttp.Response)
			ctx.Request.CopyTo(req)
			err := fasthttp.Do(req, resp)
			if err != nil {
				log.Info().Msgf("request error: %v", err.Error())
				ctx.Error(err.Error(), fasthttp.StatusServiceUnavailable)
			}
			resp.CopyTo(&ctx.Response)
		}
	}

	addr := fmt.Sprintf(":%s", port)
	log.Info().Msgf("starting proxy listener => %s", addr)

	// TODO: healthchecks and authentication middleware integration with CLI
	// log.Fatal().Msgf("terminated: %v", fasthttp.ListenAndServe(addr, pingMW(authMW(forwarder))))

	log.Fatal().Msgf("terminated: %v", fasthttp.ListenAndServe(addr, forwarder))
}
