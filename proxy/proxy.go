package proxy

import (
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

// var (
// 	username = os.Getenv("PROXY_USERNAME")
// 	password = os.Getenv("PROXY_PASSWORD")
// )

// func authMW(rh fasthttp.RequestHandler) fasthttp.RequestHandler {
// 	return func(ctx *fasthttp.RequestCtx) {
// 		auth := ctx.Request.Header.Peek("Proxy-Authorization")
// 		if len(auth) == 0 {
// 			// DENY
// 			ctx.Response.SetStatusCode(fasthttp.StatusProxyAuthRequired)
// 			ctx.Response.Header.Add("Proxy-Authenticate", "Basic realm=\"proxy.com\"")
// 		} else {
// 			encoded := strings.Split(string(auth), " ")
// 			if len(encoded) != 2 {
// 				// DENY
// 				ctx.Response.SetStatusCode(fasthttp.StatusProxyAuthRequired)
// 				ctx.Response.Header.Add("Proxy-Authenticate", "Basic realm=\"proxy.com\"")
// 			}
// 			decoded, err := base64.StdEncoding.DecodeString(encoded[1])
// 			if err != nil {
// 				log.Info().Msgf("decode err: %v", err)
// 				// DENY
// 				ctx.Response.SetStatusCode(fasthttp.StatusProxyAuthRequired)
// 				ctx.Response.Header.Add("Proxy-Authenticate", "Basic realm=\"proxy.com\"")
// 			}
// 			creds := strings.Split(string(decoded), ":")
// 			if len(creds) != 2 || creds[0] != username || creds[1] != password {
// 				// DENY
// 				ctx.Response.SetStatusCode(fasthttp.StatusProxyAuthRequired)
// 				ctx.Response.Header.Add("Proxy-Authenticate", "Basic realm=\"proxy.com\"")
// 			} else {
// 				// CONTINUE
// 				rh(ctx)
// 			}
// 		}
// 	}
// }

// func pingMW(rh fasthttp.RequestHandler) fasthttp.RequestHandler {
// 	return func(ctx *fasthttp.RequestCtx) {
// 		ping := string(ctx.Request.Header.Peek("X-Ping"))
// 		if len(ping) != 0 {
// 			// RESPOND
// 			ctx.Response.SetStatusCode(fasthttp.StatusOK)
// 		} else {
// 			// CONTINUE
// 			rh(ctx)
// 		}
// 	}
// }

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
