package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sunweiwe/kuber/pkg/log"
	"github.com/sunweiwe/kuber/pkg/utils/proxy"
	"github.com/sunweiwe/kuber/pkg/utils/route"
	"github.com/sunweiwe/kuber/pkg/utils/stream"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// https://github.com/kubernetes/apiserver/blob/master/pkg/server/config.go#L362
const (
	MaxRequestBodyBytes = 3 * 1024 * 1024
	RoutePrefix         = "/internal"

	ListOptionLabelSelector = "label-selector"
	ListOptionFieldSelector = "field-selector"
	ListOptionLimit         = "limit"
	ListOptionContinue      = "continue"
)

type ClientRest struct {
	Cli client.Client
}

func (h *ClientRest) Register(r *route.Router) {
	r.GET("/internal/{group}/{version}/{kind}/{name}", h.Get)
	r.GET("/internal/{group}/{version}/namespaces/{namespace}/{kind}/{name}", h.Get)

	r.GET("/internal/{group}/{version}/{kind}", h.List)
	r.GET("/internal/{group}/{version}/namespaces/{namespace}/{kind}", h.List)

	r.POST("/internal/{group}/{version}/{kind}", h.Create)
	r.POST("/internal/{group}/{version}/namespaces/{namespace}/{kind}", h.Create)

	r.PUT("/internal/{group}/{version}/{kind}/{name}", h.Update)
	r.PUT("/internal/{group}/{version}/namespaces/{namespace}/{kind}/{name}", h.Update)

	r.PATCH("/internal/{group}/{version}/{kind}/{name}", h.Patch)
	r.PATCH("/internal/{group}/{version}/namespaces/{namespace}/{kind}/{name}", h.Patch)

	r.DELETE("/internal/{group}/{version}/{kind}/{name}", h.Delete)
	r.DELETE("/internal/{group}/{version}/namespaces/{namespace}/{kind}/{name}", h.Delete)

	r.GET("/internal/{group}/{version}/namespaces/{namespace}/{kind}/{name}/portforward", h.PortForward)

	r.GET("/internal/core/v1/namespaces/{namespace}/{kind}/{name}:{port}/proxy/{proxypath}*", h.Proxy)
	r.GET("/internal/core/v1/namespaces/{namespace}/{kind}/{name}:{port}/proxy/", h.Proxy)
}

func (h *ClientRest) Get(c *gin.Context) {
	obj, gvk, err := h.Object(c, false)
	if err != nil {
		NotOK(c,
			apiErrors.NewInternalError(errors.New("list object is not client.ObjectList")),
		)
		return
	}
	ctx := c.Request.Context()
	if err := h.Cli.Get(ctx, client.ObjectKey{Namespace: gvk.Namespace, Name: gvk.Name}, obj); err != nil {
		NotOK(c, err)
	} else {
		OK(c, obj)
	}
}

func (h *ClientRest) List(c *gin.Context) {
	list, gvk := h.ListObject(c)

	var fieldSelector fields.Selector
	if fs := c.Query(ListOptionFieldSelector); fs != "" {
		selector, err := fields.ParseSelector(fs)
		if err != nil {
			NotOK(c, err)
			return
		}
		fieldSelector = selector
	}
	var labelSelector labels.Selector
	if ls := c.Query(ListOptionLabelSelector); ls != "" {
		selector, err := labels.Parse(c.Query(ListOptionLabelSelector))
		if err != nil {
			NotOK(c, err)
			return
		}
		labelSelector = selector
	}
	limit, _ := strconv.Atoi(c.Query(ListOptionLimit))
	ListOptions := &client.ListOptions{
		Namespace:     gvk.Namespace,
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
		Limit:         int64(limit),
		Continue:      c.Query(ListOptionContinue),
	}

	watch, _ := strconv.ParseBool(c.Query("watch"))
	if !watch {
		if err := h.Cli.List(c.Request.Context(), list, ListOptions); err != nil {
			NotOK(c, err)
			return
		}
		OK(c, list)
		return
	}

	watchableCli, ok := h.Cli.(client.WithWatch)
	if !ok {
		NotOK(c, apiErrors.NewServiceUnavailable("client not watchable"))
	}
	watcher, err := watchableCli.Watch(c.Request.Context(), list, ListOptions)
	if err != nil {
		NotOK(c, err)
		return
	}
	defer watcher.Stop()
	pusher, err := stream.StartPusher(c.Writer)
	if err != nil {
		NotOK(c, err)
		return
	}

	for {
		select {
		case e, ok := <-watcher.ResultChan():
			if !ok {
				return
			}
			if err := pusher.Push(e); err != nil {
				return
			}
		case <-c.Request.Context().Done():
			return
		}
	}
}

func (h *ClientRest) Create(c *gin.Context) {
	obj, _, err := h.Object(c, true)
	if err != nil {
		NotOK(c, err)
		return
	}
	options := &client.CreateOptions{}
	if err := h.Cli.Create(c.Request.Context(), obj, options); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, obj)
}

func (h *ClientRest) Update(c *gin.Context) {
	obj, _, err := h.Object(c, true)
	if err != nil {
		NotOK(c, err)
		return
	}
	subResource, _ := strconv.ParseBool(c.Query("subresource"))
	options := &client.UpdateOptions{}
	if subResource {
		if err := h.Cli.Status().Update(c.Request.Context(), obj, options); err != nil {
			NotOK(c, err)
			return
		}
	} else {
		if err := h.Cli.Update(c.Request.Context(), obj, options); err != nil {
			NotOK(c, err)
			return
		}
	}
	OK(c, obj)
}

const (
	PatchOptionForce = "force"
)

func (h *ClientRest) Patch(c *gin.Context) {
	obj, _, err := h.Object(c, false)
	if err != nil {
		NotOK(c, err)
		return
	}

	patchData, err := io.ReadAll(&io.LimitedReader{R: c.Request.Body, N: MaxRequestBodyBytes})
	if err != nil {
		NotOK(c, err)
		return
	}

	options := &client.PatchOptions{
		Force: func() *bool {
			if b := c.Query(PatchOptionForce); b != "" {
				bl, _ := strconv.ParseBool(b)
				return pointer.Bool(bl)
			}
			return nil
		}(),
		FieldManager: c.Query("field-manager"),
	}

	patch := client.RawPatch(types.PatchType(c.Request.Header.Get("Content-Type")), patchData)
	subResource, _ := strconv.ParseBool(c.Query("subresource"))
	if subResource {
		if err := h.Cli.Status().Patch(c.Request.Context(), obj, patch, options); err != nil {
			NotOK(c, err)
			return
		}
	} else {
		if err := h.Cli.Patch(c.Request.Context(), obj, patch, options); err != nil {
			NotOK(c, err)
			return
		}
	}
	OK(c, obj)
}

const (
	DeleteOptionDeletionPropagation = "deletion-propagation"
	DeleteOptionGracePeriod         = "grace-period-seconds"
)

func (h *ClientRest) Delete(c *gin.Context) {
	obj, _, err := h.Object(c, false)
	if err != nil {
		NotOK(c, err)
		return
	}

	options := &client.DeleteOptions{
		PropagationPolicy: func() *metav1.DeletionPropagation {
			if policy := metav1.DeletionPropagation(c.Query(DeleteOptionDeletionPropagation)); policy != "" {
				return &policy
			}
			return nil
		}(),
		GracePeriodSeconds: func() *int64 {
			if seconds := c.Query(DeleteOptionGracePeriod); seconds != "" {
				sec, _ := strconv.Atoi(seconds)
				return pointer.Int64(int64(sec))
			}
			return nil
		}(),
	}
	if err := h.Cli.Delete(c.Request.Context(), obj, options); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, obj)
}

func (h *ClientRest) PortForward(c *gin.Context) {
	gvk := h.parseGVK(c)

	if gvk.Group != "" || gvk.Version != "v1" {
		NotOK(c, fmt.Errorf("unsupported group: %s", gvk.GroupVersionKind.GroupVersion()))
		return
	}

	port, err := strconv.Atoi(c.Query("port"))
	if err != nil {
		NotOK(c, err)
		return
	}

	ctx := c.Request.Context()
	process := func() error {
		var target string
		switch gvk.Kind {
		case "Pod":
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: gvk.Name, Namespace: gvk.Namespace}}
			if err := h.Cli.Get(ctx, client.ObjectKeyFromObject(pod), pod); err != nil {
				return err
			}
			if pod.Status.Phase != corev1.PodRunning {
				return fmt.Errorf("pod %s is not running", pod.Name)
			}
			// pod: {pod-ip}.
			target = fmt.Sprintf("%s:%d", pod.Status.PodIP, port)
		case "Service", "":
			// see: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/
			// svc: {svcname}.{namespace}.svc
			target = fmt.Sprintf("%s.%s.svc:%d", gvk.Name, gvk.Namespace, port)
		}

		tcpProxy, err := proxy.NewTCPProxy(target, -1)
		if err != nil {
			return err
		}

		source, _, err := c.Writer.Hijack()
		if err != nil {
			return fmt.Errorf("unable hijack http connection: %v", err)
		}

		if err := tcpProxy.ServeConn(source); err != nil {
			log.Errorf("copy connection error: %v", err)
			// already hijacked,return nil avoid http response
			return nil
		}
		return nil
	}

	if err := process(); err != nil {
		NotOK(c, err)
		return
	}
}

func (h *ClientRest) Proxy(c *gin.Context) {
	gvk := h.parseGVK(c)

	port, err := strconv.Atoi(c.Param("port"))
	if err != nil {
		NotOK(c, err)
		return
	}
	proxyPath := "/" + c.Param("proxypath")
	ctx := c.Request.Context()

	process := func() error {
		var host string
		switch strings.ToLower(gvk.Kind) {

		case "pod":
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: gvk.Name, Namespace: gvk.Namespace}}
			if err := h.Cli.Get(ctx, client.ObjectKeyFromObject(pod), pod); err != nil {
				return err
			}
			if pod.Status.Phase != corev1.PodRunning {
				return fmt.Errorf("pod %s is not running", pod.Name)
			}
			host = fmt.Sprintf("%s:%d", pod.Status.PodIP, port)
		case "service", "":
			host = fmt.Sprintf("%s.%s.svc:%d", gvk.Name, gvk.Namespace, port)
		}

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = "http"
				req.URL.Host = host
				req.URL.Path = proxyPath
				req.Host = host
			},
		}

		proxy.ServeHTTP(c.Writer, c.Request)
		return nil
	}
	if err := process(); err != nil {
		NotOK(c, err)
		return
	}
}

func NotOK(c *gin.Context, err error) {
	statusErr := &apiErrors.StatusError{}
	if !errors.As(err, &statusErr) {
		statusErr = apiErrors.NewBadRequest(err.Error())
	}
	c.AbortWithStatusJSON(int(statusErr.Status().Code), statusErr.ErrStatus)
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

func (r *ClientRest) Object(c *gin.Context, body bool) (client.Object, GVK, error) {
	gvk := r.parseGVK(c)

	runtimeObject, err := r.Cli.Scheme().New(gvk.GroupVersionKind)
	if err != nil {
		runtimeObject = &unstructured.Unstructured{}
	}
	obj, ok := runtimeObject.(client.Object)
	if !ok {
		obj = &unstructured.Unstructured{}
	}
	if body {
		if err := json.NewDecoder(c.Request.Body).Decode(&obj); err != nil {
			return nil, gvk, apiErrors.NewBadRequest(err.Error())
		}
		defer c.Request.Body.Close()
	}
	obj.GetObjectKind().SetGroupVersionKind(gvk.GroupVersionKind)

	obj.SetNamespace(gvk.Namespace)
	if gvk.Name != "" {
		obj.SetName(gvk.Name)
	}
	return obj, gvk, nil
}

type GVK struct {
	schema.GroupVersionKind
	Namespace string
	Resource  string
	Name      string
}

func (r *ClientRest) parseGVK(c *gin.Context) GVK {
	gvk := GVK{
		GroupVersionKind: schema.GroupVersionKind{
			Group: func() string {
				if group := c.Param("group"); group != "core" {
					return group
				}
				return ""
			}(),
			Version: c.Param("version"),
			Kind:    c.Param("kind"),
		},
		Namespace: c.Param("namespace"),
		Name:      c.Param("name"),
	}
	return gvk
}

func (r *ClientRest) ListObject(c *gin.Context) (client.ObjectList, GVK) {
	gvk := r.parseGVK(c)
	if !strings.HasSuffix(gvk.Kind, "List") {
		gvk.GroupVersionKind.Kind = gvk.GroupVersionKind.Kind + "List"
	}

	runtimeList, err := r.Cli.Scheme().New(gvk.GroupVersionKind)
	if err != nil {
		runtimeList = &unstructured.UnstructuredList{}
	}
	objectList, ok := runtimeList.(client.ObjectList)
	if !ok {
		objectList = &unstructured.UnstructuredList{}
	}
	objectList.GetObjectKind().SetGroupVersionKind(gvk.GroupVersionKind)

	return objectList, gvk
}
