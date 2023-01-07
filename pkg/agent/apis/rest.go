package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sunweiwe/kuber/pkg/agent/cluster"
	"github.com/sunweiwe/kuber/pkg/log"
	"github.com/sunweiwe/kuber/pkg/utils/pagination"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type REST struct {
	client  client.Client
	cluster cluster.Interface
}

type GVK struct {
	Action string
	schema.GroupVersionKind
	Namespace     string
	Resource      string
	Name          string
	Labels        map[string]string
	LabelSelector string
}

// @Tags        Agent.V1
// @Summary     获取namespaced scope workload
// @Description 获取namespaced scope workload
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       group     path     string                               true "group"
// @Param       version   path     string                               true "version"
// @Param       resource  path     string                               true "resoruce"
// @Param       name      path     string                               true "name"
// @Param       namespace path     string                               true "namespace"
// @Success     200       {object} handlers.ResponseStruct{Data=object} "counter"
// @Router      /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name} [get]
// @Security    JWT
func (h *REST) Get(c *gin.Context) {
	obj, gvk, err := h.Object(c, false)
	if err != nil {
		NotOK(c, err)
		return
	}
	if err = h.client.Get(c.Request.Context(),
		types.NamespacedName{Namespace: gvk.Namespace, Name: gvk.Name}, obj); err != nil {
		NotOK(c, err)
	} else {
		OK(c, obj)
	}
}

// @Tags        Agent.V1
// @Summary     创建 none namespaced scope workload
// @Description 创建 none namespaced scope workload
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       group     path     string                               true "group"
// @Param       version   path     string                               true "version"
// @Param       resource  path     string                               true "resoruce"
// @Param       name      path     string                               true "name"
// @Param       namespace path     string                               true "namespace"
// @Param       data      body     object                               true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=object} "counter"
// @Router      /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name} [post]
// @Security    JWT
func (h *REST) Create(c *gin.Context) {
	obj, _, err := h.Object(c, true)
	if err != nil {
		NotOK(c, err)
		return
	}
	if err := h.client.Create(c.Request.Context(), obj); err != nil {
		log.Warnf("create object failed: %v", err)
		NotOK(c, err)
	} else {
		OK(c, obj)
	}
}

// @Tags        Agent.V1
// @Summary     创建namespaced scope workload
// @Description 创建namespaced scope workload
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       group     path     string                               true "group"
// @Param       version   path     string                               true "version"
// @Param       resource  path     string                               true "resoruce"
// @Param       name      path     string                               true "name"
// @Param       namespace path     string                               true "namespace"
// @Param       data      body     object                               true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=object} "counter"
// @Router      /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name} [put]
// @Security    JWT
func (h *REST) Update(c *gin.Context) {
	obj, _, err := h.Object(c, true)
	if err != nil {
		NotOK(c, err)
		return
	}
	if err := h.client.Update(c.Request.Context(), obj); err != nil {
		NotOK(c, err)
	} else {
		OK(c, obj)
	}
}

// @Tags        Agent.V1
// @Summary     创建namespaced scope workload
// @Description 创建namespaced scope workload
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       group     path     string                               true "group"
// @Param       version   path     string                               true "version"
// @Param       resource  path     string                               true "resoruce"
// @Param       name      path     string                               true "name"
// @Param       namespace path     string                               true "namespace"
// @Success     200       {object} handlers.ResponseStruct{Data=object} "counter"
// @Router      /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name} [delete]
// @Security    JWT
func (h *REST) Delete(c *gin.Context) {
	obj, _, err := h.Object(c, false)
	if err != nil {
		NotOK(c, err)
		return
	}
	if err := h.client.Delete(c.Request.Context(), obj); err != nil {
		NotOK(c, err)
	} else {
		OK(c, obj)
	}
}

// @Tags        Agent.V1
// @Summary     创建namespaced scope workload
// @Description 创建namespaced scope workload
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       group     path     string                               true "group"
// @Param       version   path     string                               true "version"
// @Param       resource  path     string                               true "resoruce"
// @Param       name      path     string                               true "name"
// @Param       namespace path     string                               true "namespace"
// @Param       data      body     object                               true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=object} "counter"
// @Router      /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name} [patch]
// @Security    JWT
func (h *REST) Patch(c *gin.Context) {
	obj, _, err := h.Object(c, false)
	if err != nil {
		NotOK(c, err)
		return
	}
	defer c.Request.Body.Close()

	ctx := c.Request.Context()
	var patch client.Patch
	var patchOptions []client.PatchOption
	switch patchType := types.PatchType(c.Request.Header.Get("Content_Type")); patchType {
	case types.MergePatchType, types.JSONPatchType, types.StrategicMergePatchType:
		patchData, _ := io.ReadAll(c.Request.Body)
		patch = client.RawPatch(patchType, patchData)
	case types.ApplyPatchType:
		if err := json.NewDecoder(c.Request.Body).Decode(obj); err != nil {
			NotOK(c, err)
			return
		}
		obj.SetManagedFields(nil)
		patch = client.Apply
		patchOptions = append(patchOptions, client.FieldOwner("kuber-agent"), client.ForceOwnership)
	default:
		if err := json.NewDecoder(c.Request.Body).Decode(obj); err != nil {
			NotOK(c, err)
			return
		}
		exist, _ := obj.DeepCopyObject().(client.Object)
		if err := h.client.Get(ctx, client.ObjectKeyFromObject(exist), exist); err != nil {
			NotOK(c, err)
			return
		}
		obj.SetResourceVersion(exist.GetResourceVersion())
		if err := h.client.Update(ctx, obj); err != nil {
			NotOK(c, err)
			return
		}
		OK(c, obj)
		return
	}
	if err := h.client.Patch(ctx, obj, patch, patchOptions...); err != nil {
		NotOK(c, err)
	} else {
		OK(c, obj)
	}
}

type scaleForm struct {
	Replicas int32 `json:"replicas"`
}

// @Tags        Agent.V1
// @Summary     扩缩容
// @Description 扩缩容
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                               true "cluster"
// @Param       group     path     string                               true "group"
// @Param       version   path     string                               true "version"
// @Param       resource  path     string                               true "resoruce"
// @Param       name      path     string                               true "name"
// @Param       namespace path     string                               true "namespace"
// @Param       data      body     scaleForm                            true "body"
// @Success     200       {object} handlers.ResponseStruct{Data=object} "counter"
// @Router      /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name}/actions/scale [patch]
// @Security    JWT
func (h *REST) Scale(c *gin.Context) {
	gvk, err := h.parseGVK(c)
	if err != nil {
		NotOK(c, err)
		return
	}
	formData := scaleForm{}
	if e := c.BindJSON(&formData); e != nil {
		NotOK(c, e)
		return
	}

	patch := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"replicas": formData.Replicas,
			},
		},
	}
	patch.SetGroupVersionKind(gvk.GroupVersionKind)
	patch.SetName(gvk.Name)
	patch.SetNamespace(gvk.Namespace)

	if err := h.client.Patch(c, patch, client.Merge); err != nil {
		NotOK(c, err)
	} else {
		OK(c, patch)
	}
}

// @Tags        Agent.V1
// @Summary     获取namespaced scope workload  list
// @Description 获取namespaced scope workload  list
// @Accept      json
// @Produce     json
// @Param       cluster   path     string                                 true "cluster"
// @Param       group     path     string                                 true "group"
// @Param       version   path     string                                 true "version"
// @Param       resource  path     string                                 true "resource"
// @Param       namespace path     string                                 true "namespace"
// @Param       watch     query    bool                                   true "watch"
// @Success     200       {object} handlers.ResponseStruct{Data=[]object} "counter"
// @Router      /v1/proxy/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource} [get]
// @Security    JWT
func (h *REST) List(c *gin.Context) {
	list, gvk, err := h.config(c)
	if err != nil {
		NotOK(c, err)
		return
	}
	lisOption := &client.ListOptions{
		Namespace:     gvk.Namespace,
		LabelSelector: parseLabelSelector(gvk.Labels, gvk.LabelSelector),
	}
	if err := h.client.List(c.Request.Context(), list, lisOption); err != nil {
		NotOK(c, err)
		return
	}
	items, err := ExtractList(list)
	if err != nil {
		NotOK(c, err)
		return
	}

	pageData := pagination.NewTypedSearchSortPageResourceFromContext(c, items)
	if watched, _ := strconv.ParseBool(c.Param("watch")); watched {
		c.SSEvent("data", pageData)
		c.Writer.Flush()

		return
	} else {
		OK(c, pageData)
		return
	}
}

func ExtractList(obj runtime.Object) ([]client.Object, error) {
	itemsPtr, err := meta.GetItemsPtr(obj)
	if err != nil {
		return nil, err
	}
	items, err := conversion.EnforcePtr(itemsPtr)
	if err != nil {
		return nil, err
	}
	list := make([]client.Object, items.Len())
	for i := range list {
		raw := items.Index(i)
		switch item := raw.Interface().(type) {
		case client.Object:
			list[i] = item
		default:
			var dp bool
			if list[i], dp = raw.Addr().Interface().(client.Object); !dp {
				return nil, fmt.Errorf("%v: item[%v]: Expected object, got %#v(%s)", obj, i, raw.Interface(), raw.Kind())
			}
		}
	}
	return list, nil
}

func parseLabelSelector(mapSelector map[string]string, selector string) labels.Selector {
	var selectorLabel labels.Selector
	if len(selector) > 0 {
		selectorLabel, err := labels.Parse(selector)
		if err == nil {
			return selectorLabel
		}
	}
	selectorLabel = labels.NewSelector()
	for k, v := range mapSelector {
		if !strings.Contains(k, "__") {
			if req, err := labels.NewRequirement(k, selection.Equals, []string{v}); err == nil {
				selectorLabel = selectorLabel.Add(*req)
			}
		} else {
			sep := strings.Split(k, "__")
			length := len(sep)
			key := strings.Join(sep[:length-1], "__")
			op := sep[length-1]
			switch op {
			case "exist":
				if req, err := labels.NewRequirement(key, selection.Exists, []string{}); err == nil {
					selectorLabel = selectorLabel.Add(*req)
				}
			case "neq":
				if req, err := labels.NewRequirement(key, selection.NotEquals, []string{v}); err == nil {
					selectorLabel = selectorLabel.Add(*req)
				}
			case "notexist":
				if req, err := labels.NewRequirement(key, selection.DoesNotExist, []string{}); err == nil {
					selectorLabel = selectorLabel.Add(*req)
				}
			case "in":
				if req, err := labels.NewRequirement(key, selection.In, strings.Split(v, ",")); err == nil {
					selectorLabel = selectorLabel.Add(*req)
				}
			case "notin":
				if req, err := labels.NewRequirement(key, selection.NotIn, strings.Split(v, ",")); err == nil {
					selectorLabel = selectorLabel.Add(*req)
				}

			}
		}
	}
	return selectorLabel
}

func (r *REST) parseGVK(c *gin.Context) (GVK, error) {
	group := c.Param("group")
	if group == "core" {
		group = ""
	}
	namespace := c.Param("namespace")
	if namespace == "_all" || namespace == "_" {
		namespace = ""
	}

	gvk := GVK{
		GroupVersionKind: schema.GroupVersionKind{
			Group:   group,
			Version: c.Param("version"),
		},
		Namespace:     namespace,
		Name:          c.Param("name"),
		Resource:      c.Param("resource"),
		Labels:        c.QueryMap("labels"),
		LabelSelector: c.Query("labelSelector"),
	}

	gvkn, err := r.client.RESTMapper().KindFor(gvk.GroupVersion().WithResource(gvk.Resource))
	if err != nil {
		return GVK{}, err
	}
	gvk.GroupVersionKind = gvkn
	return gvk, nil
}

func (r *REST) config(c *gin.Context) (client.ObjectList, GVK, error) {
	gvk, err := r.parseGVK(c)
	if err != nil {
		return nil, GVK{}, err
	}
	if !strings.HasPrefix(gvk.Kind, "List") {
		gvk.GroupVersionKind.Kind = gvk.GroupVersionKind.Kind + "List"
	}

	runList, err := r.client.Scheme().New(gvk.GroupVersionKind)
	if err != nil {
		runList = &unstructured.UnstructuredList{}
	}

	paramList, ok := runList.(client.ObjectList)
	if !ok {
		paramList = &unstructured.UnstructuredList{}
	}
	paramList.GetObjectKind().SetGroupVersionKind(gvk.GroupVersionKind)
	return paramList, gvk, nil
}

func WatchEvents(c *gin.Context, cluster cluster.Interface, list client.ObjectList, opts ...client.ListOption) error {
	ctx, cancelFunc := context.WithCancel(c.Request.Context())
	defer cancelFunc()

	go func() {
		<-c.Writer.CloseNotify()
		cancelFunc()
	}()

	onEvent := func(e watch.Event) error {
		c.SSEvent("data", e.Object)
		c.Writer.Flush()
		return nil
	}

	if err := cluster.Watch(ctx, list, onEvent, opts...); err != nil {
		log.WithField("watch", list.GetObjectKind().GroupVersionKind().GroupKind().String()).
			Warn(err.Error())
	}

	return nil
}

func (r *REST) Object(c *gin.Context, body bool) (client.Object, GVK, error) {
	gvk, err := r.parseGVK(c)
	if err != nil {
		return nil, GVK{}, err
	}

	runtimeObject, err := r.client.Scheme().New(gvk.GroupVersionKind)
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

	if objNs := obj.GetNamespace(); objNs != "" && objNs != gvk.Namespace {
		return obj, gvk, apiErrors.NewBadRequest(
			fmt.Sprintf("namespace in path %s is different with in body %s", gvk.Namespace, objNs),
		)
	}
	if gvk.Name != "" {
		obj.SetName(gvk.Name)
	}
	obj.GetObjectKind().SetGroupVersionKind(gvk.GroupVersionKind)
	obj.SetNamespace(gvk.Namespace)
	return obj, gvk, nil
}
