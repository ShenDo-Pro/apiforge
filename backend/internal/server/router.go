package server

import (
	"context"
	"net/http"
	"strings"
)

// 路径参数存入上下文所用的键，避免与其它包冲突。
type ctxKey int

const paramsCtxKey ctxKey = 0

// Param 从请求上下文取出路由路径参数（如 :projectID）。
func Param(r *http.Request, name string) string {
	if m, ok := r.Context().Value(paramsCtxKey).(map[string]string); ok {
		return m[name]
	}
	return ""
}

type route struct {
	segs    []string
	handler http.Handler
}

// Router 是零依赖的小型路由，支持 :param 路径参数匹配。
// 足以覆盖本项目的 REST 路由，无需引入第三方 Web 框架。
type Router struct {
	routes   map[string][]route
	notFound http.Handler
}

func NewRouter() *Router {
	return &Router{routes: map[string][]route{}}
}

// Handle 注册一条路由，pattern 形如 /api/project/:id。
func (rt *Router) Handle(method, pattern string, h http.Handler) {
	rt.routes[method] = append(rt.routes[method], route{
		segs:    splitPath(pattern),
		handler: h,
	})
}

func (rt *Router) Get(p string, h http.Handler)    { rt.Handle(http.MethodGet, p, h) }
func (rt *Router) Post(p string, h http.Handler)   { rt.Handle(http.MethodPost, p, h) }
func (rt *Router) Put(p string, h http.Handler)    { rt.Handle(http.MethodPut, p, h) }
func (rt *Router) Delete(p string, h http.Handler) { rt.Handle(http.MethodDelete, p, h) }

func (rt *Router) SetNotFound(h http.Handler) { rt.notFound = h }

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	psegs := splitPath(r.URL.Path)
	for _, rte := range rt.routes[r.Method] {
		if len(rte.segs) != len(psegs) {
			continue
		}
		params := map[string]string{}
		match := true
		for i, s := range rte.segs {
			if strings.HasPrefix(s, ":") {
				params[s[1:]] = psegs[i]
			} else if s != psegs[i] {
				match = false
				break
			}
		}
		if match {
			ctx := context.WithValue(r.Context(), paramsCtxKey, params)
			rte.handler.ServeHTTP(w, r.WithContext(ctx))
			return
		}
	}
	if rt.notFound != nil {
		rt.notFound.ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return []string{}
	}
	return strings.Split(p, "/")
}
