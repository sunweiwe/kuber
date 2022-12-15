package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sunweiwe/kuber/pkg/agent/cluster.go"
	"github.com/sunweiwe/kuber/pkg/agent/ws"
	"github.com/sunweiwe/kuber/pkg/log"
	"github.com/sunweiwe/kuber/pkg/service/handlers"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

type PodHandler struct {
	cluster      cluster.Interface
	debugoptions *DebugOptions
}

// ExecContainer 进入容器交互执行命令
// @Tags        Agent.V1
// @Summary     进入容器交互执行命令(websocket)
// @Description 进入容器交互执行命令(websocket)
// @Param       cluster   path     string true "cluster"
// @Param       namespace path     string true "namespace"
// @Param       pod       path     string true "pod"
// @Param       container query    string true "container"
// @Param       stream    query    string true  "stream must be true"
// @Param       token     query    string true  "token"
// @Param       shell     query    string false "default sh, choice(bash,ash,zsh)"
// @Success     200       {object} object "ws"
// @Router      /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/pods/{name}/actions/shell [get]
// @Security    JWT
func (h *PodHandler) ExecPods(c *gin.Context) {
	conn, err := ws.InitWebsocket(c.Writer, c.Request)
	if err != nil {
		_ = conn.WsWrite(websocket.TextMessage, []byte("init websocket connection error"))
		return
	}
	handler := &ws.StreamHandler{WsConn: conn, ResizeEvent: make(chan remotecommand.TerminalSize)}
	exec, err := h.getExec(c)
	if err != nil {
		log.Infof("Upgrade websocket Failed: %s", err.Error())
		handlers.NotOK(c, err)
		return
	}
	if err = exec.Stream(remotecommand.StreamOptions{
		Stdin:             handler,
		Stdout:            handler,
		Stderr:            handler,
		TerminalSizeQueue: handler,
		Tty:               true,
	}); err != nil {
		_ = conn.WsWrite(websocket.TextMessage, []byte(err.Error()))
		return
	}

}

func (h *PodHandler) getExec(c *gin.Context) (remotecommand.Executor, error) {
	pe := &PodCmdExecutor{
		Cluster:   h.cluster,
		Namespace: c.Param("namespace"),
		Pod:       c.Param("name"),
		Container: paramFromHeaderOrQuery(c, "container", ""),
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}
	command := []string{
		"/bin/sh",
		"-c",
		"export LINES=20; export COLUMNS=100; export LANG=C.UTF-8; export TERM=xterm-256color; [ -x /bin/bash ] && exec /bin/bash || exec /bin/sh",
	}
	return pe.executor(command)
}

type PodCmdExecutor struct {
	Cluster   cluster.Interface
	Namespace string
	Pod       string
	Container string
	Stdin     bool
	Stdout    bool
	Stderr    bool
	TTY       bool
}

func (pe *PodCmdExecutor) executor(cmd []string) (remotecommand.Executor, error) {
	req := pe.Cluster.Kubernetes().CoreV1().RESTClient().Post().Resource("pods").Namespace(pe.Namespace).
		Name(pe.Pod).SubResource("exec").VersionedParams(&v1.PodExecOptions{
		Container: pe.Container,
		Command:   cmd,
		Stdin:     pe.Stdin,
		Stdout:    pe.Stdout,
		Stderr:    pe.Stderr,
		TTY:       pe.TTY,
	}, scheme.ParameterCodec)
	return remotecommand.NewSPDYExecutor(pe.Cluster.Config(), "POST", req.URL())
}
