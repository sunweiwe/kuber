package pagination

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Pagination struct {
	Total   int64
	List    interface{}
	Current int64
	Size    int64
}

type QueryParams struct {
	Page   int    `form:"page"`
	Size   int    `form:"size"`
	Search string `form:"search"`
	Sort   string `form:"sort"`
}

const defaultPageSize = 10

type SortAndSearchAble interface {
	GetName() string
	GetCreationTimestamp() metav1.Time
}

type TypedPagination[T any] struct {
	Total   int64
	List    []T
	Current int64
	Size    int64
}

type Named interface {
	GetName() string
}

func NewTypedSearchSortPageResourceFromContext[T any](c *gin.Context, list []T) TypedPagination[T] {
	var q QueryParams
	_ = c.BindQuery(&q)
	search := func(item T) bool {
		if obj, ok := any(item).(Named); ok {
			return SearchName(q.Search)(obj)
		}
		if obj, ok := any(&item).(client.Object); ok {
			return SearchName(q.Search)(obj)
		}

		return true
	}
	sort := func(a, b T) bool {
		// if []*Pod
		objA, oka := any(a).(SortAndSearchAble)
		objB, okb := any(b).(SortAndSearchAble)
		if oka && okb {
			return ResourceSortBy(q.Sort)(objA, objB)
		}
		// if []Pod
		objA, oka = any(&a).(SortAndSearchAble)
		objB, okb = any(&b).(SortAndSearchAble)
		if oka && okb {
			return ResourceSortBy(q.Sort)(objA, objB)
		}

		return false
	}

	return NewTypedSearchSortPage(list, q.Page, q.Size, search, sort)
}

func NewTypedSearchSortPage[T any](list []T, page, size int, pickFun func(item T) bool, sortFun func(a, b T) bool) TypedPagination[T] {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defaultPageSize
	}

	if pickFun != nil {
		data := []T{}
		for _, item := range list {
			if pickFun(item) {
				data = append(data, item)
			}
		}
		list = data
	}

	if sortFun != nil {
		sort.Slice(list, func(i, j int) bool {
			return sortFun(list[i], list[j])
		})
	}

	total := len(list)
	startIdx := (page - 1) * size
	endIdx := startIdx + size
	if startIdx > total {
		startIdx = 0
		endIdx = 0
	}
	if endIdx > total {
		endIdx = total
	}
	list = list[startIdx:endIdx]
	return TypedPagination[T]{
		Total:   int64(total),
		List:    list,
		Current: int64(page),
		Size:    int64(size),
	}

}

func SearchName(search string) func(item Named) bool {
	if search == "" {
		return func(item Named) bool {
			return true
		}
	}
	return func(item Named) bool {
		if o, ok := any(item).(client.Object); ok {
			return strings.Contains(o.GetName(), search)
		}
		return true
	}
}

func ResourceSortBy(by string) func(a, b SortAndSearchAble) bool {
	switch by {
	case "createTimeAsc":
		return func(a, b SortAndSearchAble) bool {
			return a.GetCreationTimestamp().UnixNano() < b.GetCreationTimestamp().UnixNano()
		}
	case "nameAsc", "name":
		return func(a, b SortAndSearchAble) bool {
			return strings.Compare((a.GetName()), (b.GetName())) == -1
		}
	case "nameDesc":
		return func(a, b SortAndSearchAble) bool {
			return strings.Compare((a.GetName()), (b.GetName())) == 1
		}
	case "createTimeDesc", "createTime", "time":
		return func(a, b SortAndSearchAble) bool {
			return a.GetCreationTimestamp().UnixNano() > b.GetCreationTimestamp().UnixNano()
		}
	default:
		return func(a, b SortAndSearchAble) bool {
			return a.GetCreationTimestamp().UnixNano() > b.GetCreationTimestamp().UnixNano()
		}
	}

}
