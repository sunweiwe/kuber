package apis

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sunweiwe/kuber/pkg/agent/cluster"
	"github.com/sunweiwe/kuber/pkg/agent/ws"
	"github.com/sunweiwe/kuber/pkg/api/kuber"
	"github.com/sunweiwe/kuber/pkg/log"
	"github.com/sunweiwe/kuber/pkg/service/handlers"
	"github.com/sunweiwe/kuber/pkg/utils/kubertype"
	"github.com/sunweiwe/kuber/pkg/utils/pagination"
	"golang.org/x/time/rate"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodHandler struct {
	cluster      cluster.Interface
	debugoptions *DebugOptions
}

// @Tags        Agent.V1
// @Summary     获取Pod列表数据
// @Description 获取Pod列表数据
// @Accept      json
// @Produce     json
// @Param       order         query    string                                                         false "page"
// @Param       search        query    string                                                         false "search"
// @Param       page          query    int                                                            false "page"
// @Param       size          query    int                                                            false "page"
// @Param       namespace     path     string                                                         true  "namespace"
// @Param       fieldSelector query    string                                                         false "fieldSelector, 只支持podstatus={xxx}格式"
// @Param       cluster       path     string                                                         true  "cluster"
// @Param       kind          query    string                                                         false "kind(Deployment,StatefulSet,DaemonSet,Job,Node)"
// @Param       name       query    string                                                         false "name"
// @Success     200           {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]object}} "Pod"
// @Router      /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/pods [get]
// @Security    JWT
func (h *PodHandler) List(c *gin.Context) {
	ns := c.Param("namespace")
	// 网关namespace必须是kuber-gateway
	if c.Query("kind") == "TenantGateway" {
		ns = kuber.NamespaceGateway
	}
	if ns == "_all" || ns == "_" {
		ns = ""
	}

	selLabels, err := h.ControllerLabel(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	labelsMap := c.QueryMap("labels")
	for k, v := range labelsMap {
		selLabels[k] = v
	}
	sel := labels.SelectorFromSet(selLabels)

	podList := &v1.PodList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
		client.MatchingLabelsSelector{Selector: sel},
	}

	fieldSelector, exist := fieldSelector(c)
	if exist {
		opts = append(opts, client.MatchingFieldsSelector{Selector: fieldSelector})
	}

	if err = h.cluster.GetClient().List(c.Request.Context(), podList, opts...); err != nil {
		NotOK(c, err)
		return
	}

	objects := podList.Items
	objects = filterByNodeName(c, objects)
	page := pagination.NewTypedSearchSortPageResourceFromContext(c, objects)

	if watch, _ := strconv.ParseBool(c.Query("watch")); watch {
		c.SSEvent("data", page)
		c.Writer.Flush()

		WatchEvents(c, h.cluster, podList, opts...)
	} else {
		OK(c, page)
	}
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
func (h *PodHandler) Exec(c *gin.Context) {
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

func (h *PodHandler) ControllerLabel(c *gin.Context) (map[string]string, error) {
	ns := c.Params.ByName("namespace")
	namespace := ns
	if ns == allNamespace {
		namespace = v1.NamespaceAll
	}
	ret := map[string]string{}
	kind := c.Query("kind")
	name := c.Query("name")
	if len(kind) == 0 || len(name) == 0 {
		return ret, nil
	}
	switch kind {
	case kubertype.Deployment:
		dep := &appsv1.Deployment{}
		err := h.cluster.GetClient().Get(c.Request.Context(),
			types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, dep)
		if err != nil {
			return nil, err
		}
		return dep.Spec.Selector.MatchLabels, nil
	case kubertype.StatefulSet:
		statefulSet := &appsv1.StatefulSet{}
		err := h.cluster.GetClient().Get(c.Request.Context(), types.NamespacedName{
			Namespace: namespace, Name: name,
		}, statefulSet)
		if err != nil {
			return nil, err
		}
		return statefulSet.Spec.Selector.MatchLabels, nil
	case kubertype.Job:
		job := &batchv1.Job{}
		err := h.cluster.GetClient().Get(c.Request.Context(), types.NamespacedName{
			Namespace: namespace, Name: name,
		}, job)
		if err != nil {
			return nil, err
		}
		return job.Spec.Selector.MatchLabels, nil
	case kubertype.DaemonSet:
		ds := &appsv1.DaemonSet{}
		err := h.cluster.GetClient().Get(c.Request.Context(), types.NamespacedName{
			Namespace: namespace, Name: name,
		}, ds)
		if err != nil {
			return nil, err
		}
		return ds.Spec.Selector.MatchLabels, nil
	case kubertype.TenantGateway:
		return map[string]string{"app": name}, nil
	case kubertype.ModelDeployment:
		// TODO TBD
		return map[string]string{"seldon-deployment-id": name}, nil
	}
	return ret, nil
}

func filterByNodeName(c *gin.Context, pods []v1.Pod) []v1.Pod {
	kind := c.Query("kind")
	name := c.Query("name")
	if kind != "Node" || len(name) == 0 {
		return pods
	}
	var ret []v1.Pod
	for _, pod := range pods {
		if pod.Spec.NodeName == name {
			ret = append(ret, pod)
		}
	}
	return ret
}

// ContainerLogs 获取容器的stdout输出
// @Tags        Agent.V1
// @Summary     实时获取日志STDOUT输出(websocket)
// @Description 实时获取日志STDOUT输出(websocket)
// @Param       cluster   path     string true "cluster"
// @Param       namespace path     string true "namespace"
// @Param       pod       path     string true "pod"
// @Param       container query    string true "container"
// @Param       stream    query    string true  "stream must be true"
// @Param       follow    query    string true  "follow"
// @Param       tail      query    int    false "tail line (default 1000)"
// @Success     200       {object} object "ws"
// @Router      /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/pods/{name}/actions/logs [get]
// @Security    JWT
func (h *PodHandler) ContainerLogs(c *gin.Context) {
	ws, err := ws.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Infof("Upgrade Websocket failed: %s", err.Error())
		handlers.NotOK(c, err)
		return
	}
	tailInt, _ := strconv.Atoi(paramFromHeaderOrQuery(c, "tail", "1000"))
	tail := int64(tailInt)
	logOpt := &v1.PodLogOptions{
		Container: paramFromHeaderOrQuery(c, "container", ""),
		Follow:    paramFromHeaderOrQuery(c, "follow", "true") == "true",
		TailLines: &tail,
	}
	req := h.cluster.Kubernetes().CoreV1().Pods(c.Param("namespace")).GetLogs(c.Param("name"), logOpt)
	out, err := req.Stream(c.Request.Context())
	if err != nil {
		_ = ws.WriteMessage(websocket.TextMessage, []byte("init websocket stream error"))
		return
	}
	defer out.Close()
	writer := wsWriter{
		conn: ws,
	}
	_, _ = io.Copy(&writer, out)
}

type wsWriter struct {
	conn *websocket.Conn
}

func (w *wsWriter) Write(data []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

// DownloadFile 从容器下载文件
// @Tags        Agent.V1
// @Summary     从容器下载文件
// @Description 从容器下载文件
// @Param       cluster   path     string true  "cluster"
// @Param       namespace path     string true  "namespace"
// @Param       pod       path     string true  "pod"
// @Param       container query    string true  "container"
// @Param       filename  query    string true "filename"
// @Success     200       {object} object "ws"
// @Router      /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/pods/{name}/file [get]
// @Security    JWT
func (h *PodHandler) DownloadFile(c *gin.Context) {
	filename := paramFromHeaderOrQuery(c, "filename", "")
	if e := validateFilename(filename); e != nil {
		NotOK(c, e)
		return
	}
	fd := FileTransfer{
		Cluster:   h.cluster,
		Namespace: c.Param("namespace"),
		Pod:       c.Param("name"),
		Container: paramFromHeaderOrQuery(c, "container", ""),
		Filename:  filename,
	}
	if err := fd.Download(c); err != nil {
		NotOK(c, err)
		return
	}
}

// UploadFile  upload files to container
// @Tags        Agent.V1
// @Summary     upload files to container
// @Description upload files to container
// @Param       cluster   path     string true  "cluster"
// @Param       namespace path     string true  "namespace"
// @Param       pod       path     string true  "pod"
// @Param       container query    string true  "container"
// @Param       filename  query    string true "filename"
// @Success     200       {object} object "ws"
// @Router      /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/pods/{name}/upfile [post]
// @Security    JWT
func (h *PodHandler) UploadFile(c *gin.Context) {
	fd := FileTransfer{
		Cluster:   h.cluster,
		Namespace: c.Param("namespace"),
		Pod:       c.Param("name"),
		Container: paramFromHeaderOrQuery(c, "container", ""),
	}
	if err := fd.Upload(c); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, "ok")
}

func validateFilename(name string) error {
	if name == "" || name == "/" || name == "." {
		return fmt.Errorf("filename is invalid")
	}
	if !strings.HasPrefix(name, "/") {
		return fmt.Errorf("filename is invalid, please use absolute path")
	}
	fs := strings.Split(name, "/")
	for _, sep := range fs {
		if strings.Contains(sep, "..") {
			return fmt.Errorf("filename is invalid, please use absolute path")
		}
	}
	return nil
}

type FileTransfer struct {
	Cluster   cluster.Interface
	Namespace string
	Pod       string
	Container string
	Filename  string
}

func (fd *FileTransfer) Download(c *gin.Context) error {
	pe := PodCmdExecutor{
		Cluster:   fd.Cluster,
		Namespace: fd.Namespace,
		Pod:       fd.Pod,
		Container: fd.Container,
		Stdout:    true,
		Stderr:    true,
	}
	command := []string{"tar", "cf", "-", fd.Filename}
	exec, err := pe.executor(command)
	if err != nil {
		return err
	}
	c.Header("Content-Disposition",
		mime.FormatMediaType("attachment", map[string]string{
			"filename": path.Base(fd.Filename) + ".tgz",
		}),
	)
	e := exec.Stream(remotecommand.StreamOptions{
		Stdout: RateLimitWriter(context.TODO(), c.Writer, 1024*1024),
		Stderr: &fakeStdoutWriter{},
	})

	return e
}

func (fd *FileTransfer) Upload(c *gin.Context) error {
	uploadFormData := &uploadForm{}
	if err := c.Bind(uploadFormData); err != nil {
		return err
	}
	r, w := io.Pipe()
	go uploadFormData.convertTar(w)
	pe := PodCmdExecutor{
		Cluster:   fd.Cluster,
		Namespace: fd.Namespace,
		Pod:       fd.Pod,
		Container: fd.Container,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
	}
	command := []string{"tar", "xf", "-", "-C", uploadFormData.Dest}
	exec, err := pe.executor(command)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  r,
		Stdout: &fakeStdoutWriter{},
		Stderr: &fakeStdoutWriter{},
	})
}

type uploadForm struct {
	Dest  string                  `form:"dest" binding:"required"`
	Files []*multipart.FileHeader `form:"files[]" binding:"required"`
}

func (uf *uploadForm) convertTar(w io.WriteCloser) (err error) {
	tw := tar.NewWriter(w)
	for _, file := range uf.Files {
		tw.WriteHeader(&tar.Header{
			Name:    file.Filename,
			Size:    file.Size,
			ModTime: time.Now(),
			Mode:    0644,
		})
		fd, err := file.Open()
		if err != nil {
			return err
		}
		io.Copy(tw, fd)
		fd.Close()
	}
	if e := tw.Close(); e != nil {
		log.Error(e, "convert tar error")
		return e
	}
	return w.Close()
}

type fakeStdoutWriter struct{}

func (fw *fakeStdoutWriter) Write(p []byte) (int, error) {
	// TODO: handle Stderr to response info
	log.Info("File transfer stderr: ", "content", p)
	return len(p), nil
}

type rateLimitWriter struct {
	ctx          context.Context
	originWriter io.Writer
	rateLimiter  *rate.Limiter
}

func (rw *rateLimitWriter) waitUntilAvailable(n int) {
	if !rw.rateLimiter.AllowN(time.Now(), n) {
		rw.rateLimiter.WaitN(rw.ctx, n)
	}
}

func (rw *rateLimitWriter) Write(p []byte) (int, error) {
	max := rw.rateLimiter.Burst()
	pl := len(p)
	if pl > max {
		write := 0
		page := pl / max
		last := pl % max
		for i := 0; i < page; i++ {
			rw.waitUntilAvailable(max)
			t, err := rw.originWriter.Write(p[i*max : i*max+max])
			write += t
			if err != nil {
				return write, err
			}
		}
		if last != 0 {
			rw.waitUntilAvailable(last)
			t, err := rw.originWriter.Write(p[page*max : pl])
			write += t
			return write, err
		}
		return write, nil
	} else {
		rw.waitUntilAvailable(pl)
		return rw.originWriter.Write(p)
	}
}

func RateLimitWriter(ctx context.Context, w io.Writer, speed int) io.Writer {
	l := rate.NewLimiter(rate.Limit(speed), speed*2)
	return &rateLimitWriter{
		ctx:          ctx,
		originWriter: w,
		rateLimiter:  l,
	}
}
