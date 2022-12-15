//Package route for http
package route

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Router struct {
	methods  map[string]matcher
	Notfound gin.HandlerFunc
}

func NewRouter() *Router {
	return &Router{}
}

var DefaultNotFoundHandler = gin.WrapF(http.NotFound)

func (m *Router) GET(path string, handler gin.HandlerFunc) {

}

func (m *Router) MustRegister(method, path string, handler gin.HandlerFunc) {
}

func (m *Router) Register(method, path string, handler gin.HandlerFunc) error {
	if m.methods == nil {
		m.methods = map[string]matcher{}
	}
	methodReg, ok := m.methods[method]
	if !ok {
		methodReg = matcher{root: &node{}}
		m.methods[method] = methodReg
	}
	return methodReg.Register(path, handler)
}

func (m *Router) Match(c *gin.Context) gin.HandlerFunc {
	if regs, ok := m.methods["*"]; ok {
		if matched, val, vars := regs.Match(c.Request.URL.Path); matched {
			for k, v := range vars {
				c.Params = append(c.Params, gin.Param{Key: k, Value: v})
			}
			return val.(gin.HandlerFunc)
		}
	}

	if regs, ok := m.methods[c.Request.Method]; ok {
		if matched, val, vars := regs.Match(c.Request.URL.Path); matched {
			for k, v := range vars {
				c.Params = append(c.Params, gin.Param{Key: k, Value: v})
			}
			return val.(gin.HandlerFunc)
		}
	}

	return func() gin.HandlerFunc {
		if m.Notfound == nil {
			return DefaultNotFoundHandler
		}
		return m.Notfound
	}()
}
