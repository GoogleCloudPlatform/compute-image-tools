package workflow

import (
	"fmt"
	"sync"
)

type resource struct {
	name, real, link   string
	noCleanup, deleted bool
}

type resourceMap struct {
	m  map[string]*resource
	mx sync.Mutex
}

func (rm *resourceMap) add(name string, r *resource) {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	if rm.m == nil {
		rm.m = map[string]*resource{}
	}
	rm.m[name] = r
}

func (rm *resourceMap) del(name string) {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	if rm.m != nil {
		delete(rm.m, name)
	}
}

func (rm *resourceMap) get(name string) (*resource, bool) {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	if rm.m == nil {
		return nil, false
	}
	r, ok := rm.m[name]
	return r, ok
}

func initWorkflowResources(w *Workflow) {
	disks[w] = &resourceMap{}
	images[w] = &resourceMap{}
	instances[w] = &resourceMap{}
	w.addCleanupHook(resourceCleanupHook(w))
}

func resourceCleanupHook(w *Workflow) func() error {
	return func() error {
		resourceCleanupHelper(images[w], func(r *resource) error { return deleteImage(w, r) })
		resourceCleanupHelper(instances[w], func(r *resource) error { return deleteInstance(w, r) })
		resourceCleanupHelper(disks[w], func(r *resource) error { return deleteDisk(w, r) })
		return nil
	}
}

func resourceCleanupHelper(rm *resourceMap, deleteFn func(*resource) error) {
	var wg sync.WaitGroup
	for name, r := range rm.m {
		if r.noCleanup || r.deleted {
			continue
		}
		wg.Add(1)
		go func(ref string, res *resource) {
			defer wg.Done()
			if err := deleteFn(res); err != nil {
				fmt.Println(err)
			}
		}(name, r)
	}
	wg.Wait()
}
