# Kubernetes Informer

## Code

`client-go@b111a621:tools/cache/shared_informer.go:338`

### Interfaces And Structs

#### Interfaces

有两个关于 `informer` 的 `interface`

``` go
type SharedInformer interface {
    // AddEventHandler adds an event handler to the shared informer using the shared informer's resync
    // period.  Events to a single handler are delivered sequentially, but there is no coordination
    // between different handlers.
    AddEventHandler(handler ResourceEventHandler)
    // AddEventHandlerWithResyncPeriod adds an event handler to the
    // shared informer with the requested resync period; zero means
    // this handler does not care about resyncs.  The resync operation
    // consists of delivering to the handler an update notification
    // for every object in the informer's local cache; it does not add
    // any interactions with the authoritative storage.  Some
    // informers do no resyncs at all, not even for handlers added
    // with a non-zero resyncPeriod.  For an informer that does
    // resyncs, and for each handler that requests resyncs, that
    // informer develops a nominal resync period that is no shorter
    // than the requested period but may be longer.  The actual time
    // between any two resyncs may be longer than the nominal period
    // because the implementation takes time to do work and there may
    // be competing load and scheduling noise.
    AddEventHandlerWithResyncPeriod(handler ResourceEventHandler, resyncPeriod time.Duration)
    // GetStore returns the informer's local cache as a Store.
    GetStore() Store
    // GetController is deprecated, it does nothing useful
    GetController() Controller
    // Run starts and runs the shared informer, returning after it stops.
    // The informer will be stopped when stopCh is closed.
    Run(stopCh <-chan struct{})
    // HasSynced returns true if the shared informer's store has been
    // informed by at least one full LIST of the authoritative state
    // of the informer's object collection.  This is unrelated to "resync".
    HasSynced() bool
    // LastSyncResourceVersion is the resource version observed when last synced with the underlying
    // store. The value returned is not synchronized with access to the underlying store and is not
    // thread-safe.
    LastSyncResourceVersion() string
}

// SharedIndexInformer provides add and get Indexers ability based on SharedInformer.
type SharedIndexInformer interface {
    SharedInformer
    // AddIndexers add indexers to the informer before it starts.
    AddIndexers(indexers Indexers) error
    GetIndexer() Indexer
}
```

#### Structs

``` go
type sharedIndexInformer struct {
    indexer    Indexer
    controller Controller

    processor             *sharedProcessor
    cacheMutationDetector MutationDetector

    listerWatcher ListerWatcher

    // objectType is an example object of the type this informer is
    // expected to handle.  Only the type needs to be right, except
    // that when that is `unstructured.Unstructured` the object's
    // `"apiVersion"` and `"kind"` must also be right.
    objectType runtime.Object

    // resyncCheckPeriod is how often we want the reflector's resync timer to fire so it can call
    // shouldResync to check if any of our listeners need a resync.
    resyncCheckPeriod time.Duration
    // defaultEventHandlerResyncPeriod is the default resync period for any handlers added via
    // AddEventHandler (i.e. they don't specify one and just want to use the shared informer's default
    // value).
    defaultEventHandlerResyncPeriod time.Duration
    // clock allows for testability
    clock clock.Clock

    started, stopped bool
    startedLock      sync.Mutex

    // blockDeltas gives a way to stop all event distribution so that a late event handler
    // can safely join the shared informer.
    blockDeltas sync.Mutex
}

func (s *sharedIndexInformer) Run(stopCh <-chan struct{}) {
    defer utilruntime.HandleCrash()

    fifo := NewDeltaFIFOWithOptions(DeltaFIFOOptions{
        KnownObjects:          s.indexer,
        EmitDeltaTypeReplaced: true,
    })

    cfg := &Config{
        Queue:            fifo,
        ListerWatcher:    s.listerWatcher,
        ObjectType:       s.objectType,
        FullResyncPeriod: s.resyncCheckPeriod,
        RetryOnError:     false,
        ShouldResync:     s.processor.shouldResync,

        Process: s.HandleDeltas,
    }

    func() {
        s.startedLock.Lock()
        defer s.startedLock.Unlock()

        s.controller = New(cfg)
        s.controller.(*controller).clock = s.clock
        s.started = true
    }()

    // Separate stop channel because Processor should be stopped strictly after controller
    processorStopCh := make(chan struct{})
    var wg wait.Group
    defer wg.Wait()              // Wait for Processor to stop
    defer close(processorStopCh) // Tell Processor to stop
    wg.StartWithChannel(processorStopCh, s.cacheMutationDetector.Run)
    wg.StartWithChannel(processorStopCh, s.processor.run)

    defer func() {
        s.startedLock.Lock()
        defer s.startedLock.Unlock()
        s.stopped = true // Don't want any new listeners
    }()
    s.controller.Run(stopCh)
}

func (s *sharedIndexInformer) HasSynced() bool {
    s.startedLock.Lock()
    defer s.startedLock.Unlock()

    if s.controller == nil {
        return false
    }
    return s.controller.HasSynced()
}

func (s *sharedIndexInformer) LastSyncResourceVersion() string {
    s.startedLock.Lock()
    defer s.startedLock.Unlock()

    if s.controller == nil {
        return ""
    }
    return s.controller.LastSyncResourceVersion()
}

func (s *sharedIndexInformer) GetStore() Store {
    return s.indexer
}

func (s *sharedIndexInformer) GetIndexer() Indexer {
    return s.indexer
}

func (s *sharedIndexInformer) AddIndexers(indexers Indexers) error {
    s.startedLock.Lock()
    defer s.startedLock.Unlock()

    if s.started {
        return fmt.Errorf("informer has already started")
    }

    return s.indexer.AddIndexers(indexers)
}

func (s *sharedIndexInformer) GetController() Controller {
    return &dummyController{informer: s}
}

func (s *sharedIndexInformer) AddEventHandler(handler ResourceEventHandler) {
    s.AddEventHandlerWithResyncPeriod(handler, s.defaultEventHandlerResyncPeriod)
}

func determineResyncPeriod(desired, check time.Duration) time.Duration {
    if desired == 0 {
        return desired
    }
    if check == 0 {
        klog.Warningf("The specified resyncPeriod %v is invalid because this shared informer doesn't support resyncing", desired)
        return 0
    }
    if desired < check {
        klog.Warningf("The specified resyncPeriod %v is being increased to the minimum resyncCheckPeriod %v", desired, check)
        return check
    }
    return desired
}

const minimumResyncPeriod = 1 * time.Second

func (s *sharedIndexInformer) AddEventHandlerWithResyncPeriod(handler ResourceEventHandler, resyncPeriod time.Duration) {
    s.startedLock.Lock()
    defer s.startedLock.Unlock()

    if s.stopped {
        klog.V(2).Infof("Handler %v was not added to shared informer because it has stopped already", handler)
        return
    }

    if resyncPeriod > 0 {
        if resyncPeriod < minimumResyncPeriod {
            klog.Warningf("resyncPeriod %d is too small. Changing it to the minimum allowed value of %d", resyncPeriod, minimumResyncPeriod)
            resyncPeriod = minimumResyncPeriod
        }

        if resyncPeriod < s.resyncCheckPeriod {
            if s.started {
                klog.Warningf("resyncPeriod %d is smaller than resyncCheckPeriod %d and the informer has already started. Changing it to %d", resyncPeriod, s.resyncCheckPeriod, s.resyncCheckPeriod)
                resyncPeriod = s.resyncCheckPeriod
            } else {
                // if the event handler's resyncPeriod is smaller than the current resyncCheckPeriod, update
                // resyncCheckPeriod to match resyncPeriod and adjust the resync periods of all the listeners
                // accordingly
                s.resyncCheckPeriod = resyncPeriod
                s.processor.resyncCheckPeriodChanged(resyncPeriod)
            }
        }
    }

    listener := newProcessListener(handler, resyncPeriod, determineResyncPeriod(resyncPeriod, s.resyncCheckPeriod), s.clock.Now(), initialBufferSize)

    if !s.started {
        s.processor.addListener(listener)
        return
    }

    // in order to safely join, we have to
    // 1. stop sending add/update/delete notifications
    // 2. do a list against the store
    // 3. send synthetic "Add" events to the new handler
    // 4. unblock
    s.blockDeltas.Lock()
    defer s.blockDeltas.Unlock()

    s.processor.addListener(listener)
    for _, item := range s.indexer.List() {
        listener.add(addNotification{newObj: item})
    }
}

func (s *sharedIndexInformer) HandleDeltas(obj interface{}) error {
    s.blockDeltas.Lock()
    defer s.blockDeltas.Unlock()

    // from oldest to newest
    for _, d := range obj.(Deltas) {
        switch d.Type {
        case Sync, Replaced, Added, Updated:
            s.cacheMutationDetector.AddObject(d.Object)
            if old, exists, err := s.indexer.Get(d.Object); err == nil && exists {
                if err := s.indexer.Update(d.Object); err != nil {
                    return err
                }

                isSync := false
                switch {
                case d.Type == Sync:
                    // Sync events are only propagated to listeners that requested resync
                    isSync = true
                case d.Type == Replaced:
                    if accessor, err := meta.Accessor(d.Object); err == nil {
                        if oldAccessor, err := meta.Accessor(old); err == nil {
                            // Replaced events that didn't change resourceVersion are treated as resync events
                            // and only propagated to listeners that requested resync
                            isSync = accessor.GetResourceVersion() == oldAccessor.GetResourceVersion()
                        }
                    }
                }
                s.processor.distribute(updateNotification{oldObj: old, newObj: d.Object}, isSync)
            } else {
                if err := s.indexer.Add(d.Object); err != nil {
                    return err
                }
                s.processor.distribute(addNotification{newObj: d.Object}, false)
            }
        case Deleted:
            if err := s.indexer.Delete(d.Object); err != nil {
                return err
            }
            s.processor.distribute(deleteNotification{oldObj: d.Object}, false)
        }
    }
    return nil
}
```

`client-go@b111a621:tools/cache/controller.go:121`

``` go
// Run begins processing items, and will continue until a value is sent down stopCh or it is closed.
// It's an error to call Run more than once.
// Run blocks; call via go.
func (c *controller) Run(stopCh <-chan struct{}) {
    defer utilruntime.HandleCrash()
    go func() {
        <-stopCh
        c.config.Queue.Close()
    }()
    r := NewReflector(
        c.config.ListerWatcher,
        c.config.ObjectType,
        c.config.Queue,
        c.config.FullResyncPeriod,
    )
    r.ShouldResync = c.config.ShouldResync
    r.clock = c.clock

    c.reflectorMutex.Lock()
    c.reflector = r
    c.reflectorMutex.Unlock()

    var wg wait.Group
    defer wg.Wait()

    // call Reflector.Run
    wg.StartWithChannel(stopCh, r.Run)

    wait.Until(c.processLoop, time.Second, stopCh)
}

// Run repeatedly uses the reflector's ListAndWatch to fetch all the
// objects and subsequent deltas.
// Run will exit when stopCh is closed.
func (r *Reflector) Run(stopCh <-chan struct{}) {
    wait.BackoffUntil(func() {
        if err := r.ListAndWatch(stopCh); err != nil {
            utilruntime.HandleError(err)
        }
    }, r.backoffManager, true, stopCh)
}
```

`c.config.Queue` 实现了 `cache.Store` 接口。在此处的实现为 `cache.DeltaFIFO`，源码位于 `tools/cache/shared_informer.go:341`

``` go
    fifo := NewDeltaFIFOWithOptions(DeltaFIFOOptions{
        KnownObjects:          s.indexer,
        EmitDeltaTypeReplaced: true,
    })

    cfg := &Config{
        Queue:            fifo,
        ListerWatcher:    s.listerWatcher,
        ObjectType:       s.objectType,
        FullResyncPeriod: s.resyncCheckPeriod,
        RetryOnError:     false,
        ShouldResync:     s.processor.shouldResync,

        Process: s.HandleDeltas,
    }
```

``` go
// ListAndWatch first lists all items and get the resource version at the moment of call,
// and then use the resource version to watch.
// It returns error if ListAndWatch didn't even try to initialize watch.
func (r *Reflector) ListAndWatch(stopCh <-chan struct{}) error {
    var resourceVersion string

    options := metav1.ListOptions{ResourceVersion: r.relistResourceVersion()}

    if err := func() error {
        initTrace := trace.New("Reflector ListAndWatch", trace.Field{"name", r.name})
        defer initTrace.LogIfLong(10 * time.Second)
        var list runtime.Object
        var paginatedResult bool
        var err error
        listCh := make(chan struct{}, 1)
        panicCh := make(chan interface{}, 1)
        go func() {
            defer func() {
                if r := recover(); r != nil {
                    panicCh <- r
                }
            }()
            // Attempt to gather list in chunks, if supported by listerWatcher, if not, the first
            // list request will return the full response.
            pager := pager.New(pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
                return r.listerWatcher.List(opts)
            }))
            switch {
            case r.WatchListPageSize != 0:
                pager.PageSize = r.WatchListPageSize
            case r.paginatedResult:
                // We got a paginated result initially. Assume this resource and server honor
                // paging requests (i.e. watch cache is probably disabled) and leave the default
                // pager size set.
            case options.ResourceVersion != "" && options.ResourceVersion != "0":
                // User didn't explicitly request pagination.
                //
                // With ResourceVersion != "", we have a possibility to list from watch cache,
                // but we do that (for ResourceVersion != "0") only if Limit is unset.
                // To avoid thundering herd on etcd (e.g. on master upgrades), we explicitly
                // switch off pagination to force listing from watch cache (if enabled).
                // With the existing semantic of RV (result is at least as fresh as provided RV),
                // this is correct and doesn't lead to going back in time.
                //
                // We also don't turn off pagination for ResourceVersion="0", since watch cache
                // is ignoring Limit in that case anyway, and if watch cache is not enabled
                // we don't introduce regression.
                pager.PageSize = 0
            }

            list, paginatedResult, err = pager.List(context.Background(), options)
            if isExpiredError(err) {
                r.setIsLastSyncResourceVersionExpired(true)
                // Retry immediately if the resource version used to list is expired.
                // The pager already falls back to full list if paginated list calls fail due to an "Expired" error on
                // continuation pages, but the pager might not be enabled, or the full list might fail because the
                // resource version it is listing at is expired, so we need to fallback to resourceVersion="" in all
                // to recover and ensure the reflector makes forward progress.
                list, paginatedResult, err = pager.List(context.Background(), metav1.ListOptions{ResourceVersion: r.relistResourceVersion()})
            }
            close(listCh)
        }()
        select {
        case <-stopCh:
            return nil
        case r := <-panicCh:
            panic(r)
        case <-listCh:
        }
        if err != nil {
            return fmt.Errorf("%s: Failed to list %v: %v", r.name, r.expectedTypeName, err)
        }

        // We check if the list was paginated and if so set the paginatedResult based on that.
        // However, we want to do that only for the initial list (which is the only case
        // when we set ResourceVersion="0"). The reasoning behind it is that later, in some
        // situations we may force listing directly from etcd (by setting ResourceVersion="")
        // which will return paginated result, even if watch cache is enabled. However, in
        // that case, we still want to prefer sending requests to watch cache if possible.
        //
        // Paginated result returned for request with ResourceVersion="0" mean that watch
        // cache is disabled and there are a lot of objects of a given type. In such case,
        // there is no need to prefer listing from watch cache.
        if options.ResourceVersion == "0" && paginatedResult {
            r.paginatedResult = true
        }

        r.setIsLastSyncResourceVersionExpired(false) // list was successful
        initTrace.Step("Objects listed")
        listMetaInterface, err := meta.ListAccessor(list)
        if err != nil {
            return fmt.Errorf("%s: Unable to understand list result %#v: %v", r.name, list, err)
        }
        resourceVersion = listMetaInterface.GetResourceVersion()
        initTrace.Step("Resource version extracted")
        items, err := meta.ExtractList(list)
        if err != nil {
            return fmt.Errorf("%s: Unable to understand list result %#v (%v)", r.name, list, err)
        }
        initTrace.Step("Objects extracted")
        if err := r.syncWith(items, resourceVersion); err != nil {
            return fmt.Errorf("%s: Unable to sync list result: %v", r.name, err)
        }
        initTrace.Step("SyncWith done")
        r.setLastSyncResourceVersion(resourceVersion)
        initTrace.Step("Resource version updated")
        return nil
    }(); err != nil {
        return err
    }

    resyncerrc := make(chan error, 1)
    cancelCh := make(chan struct{})
    defer close(cancelCh)
    go func() {
        resyncCh, cleanup := r.resyncChan()
        defer func() {
            cleanup() // Call the last one written into cleanup
        }()
        for {
            select {
            case <-resyncCh:
            case <-stopCh:
                return
            case <-cancelCh:
                return
            }
            if r.ShouldResync == nil || r.ShouldResync() {
                klog.V(4).Infof("%s: forcing resync", r.name)
                if err := r.store.Resync(); err != nil {
                    resyncerrc <- err
                    return
                }
            }
            cleanup()
            resyncCh, cleanup = r.resyncChan()
        }
    }()

    for {
        // give the stopCh a chance to stop the loop, even in case of continue statements further down on errors
        select {
        case <-stopCh:
            return nil
        default:
        }

        timeoutSeconds := int64(minWatchTimeout.Seconds() * (rand.Float64() + 1.0))
        options = metav1.ListOptions{
            ResourceVersion: resourceVersion,
            // We want to avoid situations of hanging watchers. Stop any wachers that do not
            // receive any events within the timeout window.
            TimeoutSeconds: &timeoutSeconds,
            // To reduce load on kube-apiserver on watch restarts, you may enable watch bookmarks.
            // Reflector doesn't assume bookmarks are returned at all (if the server do not support
            // watch bookmarks, it will ignore this field).
            AllowWatchBookmarks: true,
        }

        w, err := r.listerWatcher.Watch(options)
        if err != nil {
            switch {
            case isExpiredError(err):
                // Don't set LastSyncResourceVersionExpired - LIST call with ResourceVersion=RV already
                // has a semantic that it returns data at least as fresh as provided RV.
                // So first try to LIST with setting RV to resource version of last observed object.
                klog.V(4).Infof("%s: watch of %v closed with: %v", r.name, r.expectedTypeName, err)
            case err == io.EOF:
                // watch closed normally
            case err == io.ErrUnexpectedEOF:
                klog.V(1).Infof("%s: Watch for %v closed with unexpected EOF: %v", r.name, r.expectedTypeName, err)
            default:
                utilruntime.HandleError(fmt.Errorf("%s: Failed to watch %v: %v", r.name, r.expectedTypeName, err))
            }
            // If this is "connection refused" error, it means that most likely apiserver is not responsive.
            // It doesn't make sense to re-list all objects because most likely we will be able to restart
            // watch where we ended.
            // If that's the case wait and resend watch request.
            if utilnet.IsConnectionRefused(err) {
                time.Sleep(time.Second)
                continue
            }
            return nil
        }

        if err := r.watchHandler(w, &resourceVersion, resyncerrc, stopCh); err != nil {
            if err != errorStopRequested {
                switch {
                case isExpiredError(err):
                    // Don't set LastSyncResourceVersionExpired - LIST call with ResourceVersion=RV already
                    // has a semantic that it returns data at least as fresh as provided RV.
                    // So first try to LIST with setting RV to resource version of last observed object.
                    klog.V(4).Infof("%s: watch of %v closed with: %v", r.name, r.expectedTypeName, err)
                default:
                    klog.Warningf("%s: watch of %v ended with: %v", r.name, r.expectedTypeName, err)
                }
            }
            return nil
        }
    }
}
```

`Resync` 的源码为 `tools/cache/delta_fifo.go:592`

``` go
// Resync adds, with a Sync type of Delta, every object listed by
// `f.knownObjects` whose key is not already queued for processing.
// If `f.knownObjects` is `nil` then Resync does nothing.
func (f *DeltaFIFO) Resync() error {
    f.lock.Lock()
    defer f.lock.Unlock()

    if f.knownObjects == nil {
        return nil
    }

    keys := f.knownObjects.ListKeys()
    for _, k := range keys {
        if err := f.syncKeyLocked(k); err != nil {
            return err
        }
    }
    return nil
}
```

## Custom Informer

`Kubernetes` 使用 [informer-gen](https://github.com/kubernetes/code-generator/tree/master/cmd/informer-gen) 生成代码。
