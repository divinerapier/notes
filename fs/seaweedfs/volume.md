# Volume Service

## Start Service

启动 `volume service` 的代码在 `weed/volume.go` 中的 `runVolume` 函数。

``` go
func runVolume(cmd *Command, args []string) bool {

	weed_server.LoadConfiguration("security", false)

	if *v.maxCpu < 1 {
		*v.maxCpu = runtime.NumCPU()
	}
	runtime.GOMAXPROCS(*v.maxCpu)
	util.SetupProfiling(*v.cpuProfile, *v.memProfile)

	v.startVolumeServer(*volumeFolders, *maxVolumeCounts, *volumeWhiteListOption)

	return true
}
```

> 参数 `volumeFolders`: 存储 `volume` 的目标目录;  
> 参数 `maxVolumeCounts`: 每个目录最多 `volume` 数量;
> 参数 `volumeWhiteListOption`: 控制访问白名单。

`startVolumeServer` 函数如下:

``` go

func (v VolumeServerOptions) startVolumeServer(volumeFolders, maxVolumeCounts, volumeWhiteListOption string) {

	//Set multiple folders and each folder's max volume count limit'
	v.folders = strings.Split(volumeFolders, ",")
	maxCountStrings := strings.Split(maxVolumeCounts, ",")
	for _, maxString := range maxCountStrings {
		max, _ := strconv.Atoi(maxString)
		v.folderMaxLimits = append(v.folderMaxLimits, max)
	}

	//security related white list configuration
	if volumeWhiteListOption != "" {
		v.whiteList = strings.Split(volumeWhiteListOption, ",")
	}
  
	isSeperatedPublicPort := *v.publicPort != *v.port

	volumeMux := http.NewServeMux()
	publicVolumeMux := volumeMux
	if isSeperatedPublicPort {
		publicVolumeMux = http.NewServeMux()
	}

	// 索引存储类型
	volumeNeedleMapKind := storage.NeedleMapInMemory
	switch *v.indexType {
	case "leveldb":
		volumeNeedleMapKind = storage.NeedleMapLevelDb
	case "leveldbMedium":
		volumeNeedleMapKind = storage.NeedleMapLevelDbMedium
	case "leveldbLarge":
		volumeNeedleMapKind = storage.NeedleMapLevelDbLarge
	}

	masters := *v.masters

	// 读取数据，创建服务，后台心跳线程
	volumeServer := weed_server.NewVolumeServer(volumeMux, publicVolumeMux,
		*v.ip, *v.port, *v.publicUrl,
		v.folders, v.folderMaxLimits,
		volumeNeedleMapKind,
		strings.Split(masters, ","), *v.pulseSeconds, *v.dataCenter, *v.rack,
		v.whiteList,
		*v.fixJpgOrientation, *v.readRedirect,
	)

	listeningAddress := *v.bindIp + ":" + strconv.Itoa(*v.port)
	listener, _ := util.NewListener(listeningAddress, time.Duration(*v.idleConnectionTimeout)*time.Second)
	if isSeperatedPublicPort {
		publicListeningAddress := *v.bindIp + ":" + strconv.Itoa(*v.publicPort)
		publicListener, _ := util.NewListener(publicListeningAddress, time.Duration(*v.idleConnectionTimeout)*time.Second)
		go func() {
			http.Serve(publicListener, publicVolumeMux)
		}()
	}

	util.OnInterrupt(func() {
		volumeServer.Shutdown()
		pprof.StopCPUProfile()
	})

	// starting grpc server
	grpcPort := *v.port + 10000
	grpcL, _ := util.NewListener(*v.bindIp+":"+strconv.Itoa(grpcPort), 0)
	grpcS := util.NewGrpcServer(security.LoadServerTLS(viper.Sub("grpc"), "volume"))
	volume_server_pb.RegisterVolumeServerServer(grpcS, volumeServer)
	reflection.Register(grpcS)
	go grpcS.Serve(grpcL)

	if viper.GetString("https.volume.key") != "" {
		if e := http.ServeTLS(listener, volumeMux,
			viper.GetString("https.volume.cert"), viper.GetString("https.volume.key")); e != nil {
			glog.Fatalf("Volume server fail to serve: %v", e)
		}
	} else {
		if e := http.Serve(listener, volumeMux); e != nil {
			glog.Fatalf("Volume server fail to serve: %v", e)
		}
	}

}
```

在这个函数内包含了，读取配置，生成启动参数，定义索引类型，启动 `http/https` 及 `grpc` 服务。`http` 服务的 `API` 由 `adminMux` 及 `publicMux` 组成，在函数 `NewVolumeServer` 中定义。

``` go

func NewVolumeServer(adminMux, publicMux *http.ServeMux, ip string,
	port int, publicUrl string,
	folders []string, maxCounts []int,
	needleMapKind storage.NeedleMapType,
	masterNodes []string, pulseSeconds int,
	dataCenter string, rack string,
	whiteList []string,
	fixJpgOrientation bool,
	readRedirect bool) *VolumeServer {

	v := viper.GetViper()
	signingKey := v.GetString("jwt.signing.key")
	enableUiAccess := v.GetBool("access.ui")
	metrics.VolumeRegisterMetrics()

	vs := &VolumeServer{
		pulseSeconds:      pulseSeconds,
		dataCenter:        dataCenter,
		rack:              rack,
		needleMapKind:     needleMapKind,
		FixJpgOrientation: fixJpgOrientation,
		ReadRedirect:      readRedirect,
		grpcDialOption:    security.LoadClientTLS(viper.Sub("grpc"), "volume"),
	}
	vs.MasterNodes = masterNodes
	// 读取本地数据
	vs.store = storage.NewStore(port, ip, publicUrl, folders, maxCounts, vs.needleMapKind)
	// 过滤请求白名单
	vs.guard = security.NewGuard(whiteList, signingKey)

	handleStaticResources(adminMux)
	if signingKey == "" || enableUiAccess {
		// only expose the volume server details for safe environments
		adminMux.HandleFunc("/ui/index.html", vs.uiStatusHandler)
		adminMux.HandleFunc("/status", vs.guard.WhiteList(vs.statusHandler))
		adminMux.HandleFunc("/metrics", vs.guard.WhiteList(vs.metricsHandler))
	}
	// 主要 HTTP API
	adminMux.HandleFunc("/", vs.privateStoreHandler)
	if publicMux != adminMux {
		// separated admin and public port
		handleStaticResources(publicMux)
		publicMux.HandleFunc("/", vs.publicReadOnlyHandler)
	}

	// 心跳检测
	go vs.heartbeat()
	hostAddress := fmt.Sprintf("%s:%d", ip, port)
	go stats.LoopPushingMetric("volumeServer", hostAddress, stats.VolumeServerGather,
		func() (string, int) {
			return vs.MetricsAddress, vs.MetricsIntervalSec
		})
	return vs
}
```

函数 `NewVolumeServer` 包含：  
1. 读取本地已有 `volume`： `storage.NewStore`;
2. 构建主要服务 `HTTP API`
	``` go
	adminMux.HandleFunc("/", vs.privateStoreHandler)
	if publicMux != adminMux {
		// separated admin and public port
		handleStaticResources(publicMux)
		publicMux.HandleFunc("/", vs.publicReadOnlyHandler)
	}
	```
3. 发送心跳包到 `master service`: `go vs.heartbeat()`


## Load Volume

`seaweedfs` 中有关文件系统的代码都在 `weed/storage` 目录中，从 `weed/storage/store.go` 的 `NewStore` 函数开始。

``` go

// NewStore 后端存储对象管理者
// publicUrl: 上传使用
// dirnames: 存储数据目录
// maxVolumeCounts: 目录下最大 volume 数
// needleMapKind: 索引存储类型(默认使用内存)
func NewStore(port int, ip, publicUrl string, dirnames []string, maxVolumeCounts []int, needleMapKind NeedleMapType) (s *Store) {
	s = &Store{Port: port, Ip: ip, PublicUrl: publicUrl, NeedleMapType: needleMapKind}
	s.Locations = make([]*DiskLocation, 0)
	for i := 0; i < len(dirnames); i++ {
		// Location 表示对一个目录的抽象，每个目录可以有多个 volume (数量由 maxVolumeCounts 限制)
		location := NewDiskLocation(dirnames[i], maxVolumeCounts[i])
		// 加载已有数据
		location.loadExistingVolumes(needleMapKind)
		s.Locations = append(s.Locations, location)
	}
	s.NewVolumeIdChan = make(chan VolumeId, 3)
	s.DeletedVolumeIdChan = make(chan VolumeId, 3)
	return
}
```

``` go
func NewDiskLocation(dir string, maxVolumeCount int) *DiskLocation {
	location := &DiskLocation{Directory: dir, MaxVolumeCount: maxVolumeCount}
	location.volumes = make(map[VolumeId]*Volume)
	return location
}
```

``` go
func (l *DiskLocation) loadExistingVolumes(needleMapKind NeedleMapType) {
	l.Lock()
	defer l.Unlock()

	l.concurrentLoadingVolumes(needleMapKind, 10)

	glog.V(0).Infoln("Store started on dir:", l.Directory, "with", len(l.volumes), "volumes", "max", l.MaxVolumeCount)
}
```

``` go
func (l *DiskLocation) concurrentLoadingVolumes(needleMapKind NeedleMapType, concurrency int) {

	task_queue := make(chan os.FileInfo, 10*concurrency)
	go func() {
		if dirs, err := ioutil.ReadDir(l.Directory); err == nil {
			for _, dir := range dirs { // 称为 entry 更合适，这里既包含目录(dir)也包含文件(regular file)
				task_queue <- dir
			}
		}
		close(task_queue)
	}()

	var wg sync.WaitGroup
	var mutex sync.RWMutex
	for workerNum := 0; workerNum < concurrency; workerNum++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for dir := range task_queue {
				l.loadExistingVolume(dir, needleMapKind, &mutex)
			}
		}()
	}
	wg.Wait()

}
```

``` go
func (l *DiskLocation) loadExistingVolume(dir os.FileInfo, needleMapKind NeedleMapType, mutex *sync.RWMutex) {
	name := dir.Name()
	if !dir.IsDir() && strings.HasSuffix(name, ".dat") {
		// volumeIdFromPath: (不展开该函数了) filter 保留 .dat 类型的文件 
		// (volume 有两种类型文件 .data(数据)， .idx(索引))
		vid, collection, err := l.volumeIdFromPath(dir)
		if err == nil {
			mutex.RLock()
			_, found := l.volumes[vid]
			mutex.RUnlock()
			if !found {
				if v, e := NewVolume(l.Directory, collection, vid, needleMapKind, nil, nil, 0); e == nil {
					mutex.Lock()
					l.volumes[vid] = v
					mutex.Unlock()
					glog.V(0).Infof("data file %s, replicaPlacement=%s v=%d size=%d ttl=%s",
						l.Directory+"/"+name, v.ReplicaPlacement, v.Version(), v.Size(), v.Ttl.String())
				} else {
					glog.V(0).Infof("new volume %s error %s", name, e)
				}
			}
		}
	}
}
```

`NewVolume` 函数才是真正加载一个 `volume` 的地方。

``` go
// dirname: volume 物理文件夹
// collection: 可以理解为 bucket
func NewVolume(dirname string, collection string, id VolumeId, needleMapKind NeedleMapType, replicaPlacement *ReplicaPlacement, ttl *TTL, preallocate int64) (v *Volume, e error) {
	// if replicaPlacement is nil, the superblock will be loaded from disk
	v = &Volume{dir: dirname, Collection: collection, Id: id}
	v.SuperBlock = SuperBlock{ReplicaPlacement: replicaPlacement, Ttl: ttl}
	v.needleMapKind = needleMapKind
	e = v.load(true /*alsoLoadIndex*/, true /*createDatIfMissing*/, needleMapKind, preallocate)
	return
}
```
