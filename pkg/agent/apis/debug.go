package apis

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sunweiwe/kuber/pkg/agent/ws"
	"github.com/sunweiwe/kuber/pkg/service/handlers"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DebugAgentNamespace = "debug-tools"
	DebugAgentImage     = "kuber/debug-agent:latest"
	DebugToolsImage     = "kuber/debug-tools:latest"
)

// ExecContainer 调试容器(websocket)
// @Tags        Agent.V1
// @Summary     调试容器(websocket)
// @Description 调试容器(websocket)
// @Param       cluster    path     string true  "cluster"
// @Param       namespace  path     string true  "namespace"
// @Param       name       path     string true  "pod name"
// @Param       container  query    string true  "container"
// @Param       stream     query    string true  "must be true"
// @Param       agent      query    string false "agent"
// @Param       debug      query    string false "debug"
// @Param       fork       query    string false "fork"
// @Success     200        {object} object "ws"
// @Router      /v1/proxy/cluster/{cluster}/custom/core/v1/{namespace}/pods/{name}/actions/debug [get]
// @Security    JWT
func (h *PodHandler) DebugPod(c *gin.Context) {
	conn, err := ws.InitWebsocket(c.Writer, c.Request)
	if err != nil {
		_ = conn.WsWrite(websocket.TextMessage, []byte("init websocket connection error"))
		conn.WsClose()
		return
	}
	handler := &ws.StreamHandler{WsConn: conn, ResizeEvent: make(chan remotecommand.TerminalSize)}
	exec, err := h.debug(c)
	if err != nil {
		log.Printf("Upgrade Websocket failed: %s", err.Error())
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
		_ = conn.WsWrite(websocket.TextMessage, []byte("init websocket stream error"+err.Error()))
		<-time.AfterFunc(time.Duration(3)*time.Second, func() {
			conn.WsClose()
		}).C
		return
	}
}

func (h *PodHandler) debug(c *gin.Context) (remotecommand.Executor, error) {
	image := paramFromHeaderOrQuery(c, "debugimage", h.debugoptions.Image)
	command := []string{
		"kubectl",
		"-n",
		c.Param("namespace"),
		"debug",
		c.Param("name"),
		image,
		"--image-pull-policy=IfNotPresent",
		"-it",
		"--",
		"/start.sh",
	}
	podName, err := kubectlContainer(c.Request.Context(), h.cluster.GetClient(), h.debugoptions)
	if err != nil {
		return nil, err
	}
	pe := PodCmdExecutor{
		Cluster:   h.cluster,
		Namespace: h.debugoptions.Namespace,
		Pod:       podName,
		Container: "",
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}
	return pe.executor(command)
}

func kubectlContainer(ctx context.Context, ctl client.Client, debug *DebugOptions) (string, error) {
	namespace := debug.Namespace

	podList := &v1.PodList{}
	sel, err := labels.Parse(debug.PodSelector)
	if err != nil {
		return "", err
	}

	if err := ctl.List(ctx, podList, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: sel}); err != nil {
		return "", fmt.Errorf("Failed to get kubectl container %v", err)
	}
	if len(podList.Items) == 0 {
		return "", fmt.Errorf("Failed to get kubectl container")
	}

	var podName string
	randTime := rand.New(rand.NewSource(time.Now().Unix()))
	randTime.Shuffle(len(podList.Items), func(i, j int) {
		podList.Items[i], podList.Items[j] = podList.Items[j], podList.Items[i]
	})
	for _, po := range podList.Items {
		if po.Status.Phase == v1.PodRunning {
			podName = po.GetName()
			break
		}
	}

	if len(podName) == 0 {
		return podName, fmt.Errorf("Can't find kubectl container")
	}
	return podName, nil
}
