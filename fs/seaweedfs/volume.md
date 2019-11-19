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
// collection: 可以理解为 bucket, 会作为 `volume` 文件名的一部分。
func NewVolume(dirname string, collection string, id VolumeId, needleMapKind NeedleMapType, replicaPlacement *ReplicaPlacement, ttl *TTL, preallocate int64) (v *Volume, e error) {
	// if replicaPlacement is nil, the superblock will be loaded from disk
	v = &Volume{dir: dirname, Collection: collection, Id: id}
	v.SuperBlock = SuperBlock{ReplicaPlacement: replicaPlacement, Ttl: ttl}
	v.needleMapKind = needleMapKind
	e = v.load(true /*alsoLoadIndex*/, true /*createDatIfMissing*/, needleMapKind, preallocate)
	return
}
```

``` go
func (v *Volume) load(alsoLoadIndex bool, createDatIfMissing bool, needleMapKind NeedleMapType, preallocate int64) error {
	var e error
	fileName := v.FileName()
	alreadyHasSuperBlock := false

	// 校验文件存在，权限等
	if exists, canRead, canWrite, modifiedTime, fileSize := checkFile(fileName + ".dat"); exists {
		if !canRead {
			return fmt.Errorf("cannot read Volume Data file %s.dat", fileName)
		}
		if canWrite {
			v.dataFile, e = os.OpenFile(fileName+".dat", os.O_RDWR|os.O_CREATE, 0644)
			v.lastModifiedTime = uint64(modifiedTime.Unix())
		} else {
			v.dataFile, e = os.Open(fileName + ".dat")
			v.readOnly = true
		}
		// 假设，文件大小 > 8Bytes 时，包含 SuperBlock(因为 SuperBlock 的大小为 8Bytes, 创建时会)
		if fileSize >= _SuperBlockSize /*8Bytes*/ {
			alreadyHasSuperBlock = true
		}
	} else {
		if createDatIfMissing {
			v.dataFile, e = createVolumeFile(fileName+".dat", preallocate)
		} else {
			return fmt.Errorf("Volume Data file %s.dat does not exist.", fileName)
		}
	}

	if alreadyHasSuperBlock {
		e = v.readSuperBlock()
	} else {
		e = v.maybeWriteSuperBlock()
	}
	if e == nil && alsoLoadIndex {
		var indexFile *os.File
		if v.readOnly {
			indexFile, _ = os.OpenFile(fileName+".idx", os.O_RDONLY, 0644)
		} else {
			indexFile, _ = os.OpenFile(fileName+".idx", os.O_RDWR|os.O_CREATE, 0644)
		}
		// 校验索引文件
		if e = CheckVolumeDataIntegrity(v, indexFile); e != nil {
			v.readOnly = true
		}
		switch needleMapKind {
		case NeedleMapInMemory: // 默认类型
			glog.V(0).Infoln("loading index", fileName+".idx", "to memory readonly", v.readOnly)
			// 加载索引文件
			v.nm, _ = LoadCompactNeedleMap(indexFile)
		case NeedleMapLevelDb:
			opts := &opt.Options{
				BlockCacheCapacity: 2 * 1024 * 1024, // default value is 8MiB
				WriteBuffer:        1 * 1024 * 1024, // default value is 4MiB
			}
			v.nm, _ = NewLevelDbNeedleMap(fileName+".ldb", indexFile, opts)
		case NeedleMapLevelDbMedium:
			opts := &opt.Options{
				BlockCacheCapacity: 4 * 1024 * 1024, // default value is 8MiB
				WriteBuffer:        2 * 1024 * 1024, // default value is 4MiB
			}
			v.nm, _ = NewLevelDbNeedleMap(fileName+".ldb", indexFile, opts)
		case NeedleMapLevelDbLarge:
			opts := &opt.Options{
				BlockCacheCapacity: 8 * 1024 * 1024, // default value is 8MiB
				WriteBuffer:        4 * 1024 * 1024, // default value is 4MiB
			}
			v.nm, _ = NewLevelDbNeedleMap(fileName+".ldb", indexFile, opts)
		}
	}

	return e
}
```

``` go

func CheckVolumeDataIntegrity(v *Volume, indexFile *os.File) error {
	var indexSize int64
	var e error
	// 校验长度使用合法 (因为 index 的每一条记录为定长(16 Bytes)，所以文件大小应为16的整数倍)
	if indexSize, e = verifyIndexFileIntegrity(indexFile); e != nil {
		return fmt.Errorf("verifyIndexFileIntegrity %s failed: %v", indexFile.Name(), e)
	}
	if indexSize == 0 {
		return nil
	}
	var lastIdxEntry []byte
	// 读取最后一个 index 的内容, // IdxFileEntry 和 readIndexEntryAtOffset 应该作为一个函数
	if lastIdxEntry, e = readIndexEntryAtOffset(indexFile, indexSize-NeedleEntrySize); e != nil {
		return fmt.Errorf("readLastIndexEntry %s failed: %v", indexFile.Name(), e)
	}
	// key:    type = NeedleId, size = 8,  [0, 8)
	// offset: type = Offset,   size = 4   [8, 12)  表示以 8 Bytes(对齐单位) 为单位的偏移量
	// size:   type = uint32,   size = 4   [12, 16) 表示 NeedleBody 的实际大小，实际读取大小为 Header + Body + 填充
	key, offset, size := IdxFileEntry(lastIdxEntry)
	// 第一个 Needle 的 Offset 应该为 8 (SuperBlock)
	if offset.IsZero() || size == TombstoneFileSize {
		return nil
	}
	// 检验最后一个needle的数据
	if e = verifyNeedleIntegrity(v.dataFile, v.Version(), offset.ToAcutalOffset(), key, size); e != nil {
		return fmt.Errorf("verifyNeedleIntegrity %s failed: %v", indexFile.Name(), e)
	}

	return nil
}
```

## HTTP API

`Seaweedfs` 有关服务处理相关的代码在 `weed/server` 目录中。 `volume api` 的入口在 `weed/server/volume_server_handlers.go` 文件中。 

### Write

``` go
func (vs *VolumeServer) privateStoreHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if _, exists := volumeServerPrivateStoreHandlerSupportMethods[r.Method]; !exists {
		return
	}
	stats.VolumeServerRequestCounter.WithLabelValues(r.Method).Inc()
	switch r.Method {
	case "GET", "HEAD": // 读文件
		stats.ReadRequest()
		vs.GetOrHeadHandler(w, r)
	case "DELETE":
		stats.DeleteRequest()
		vs.guard.WhiteList(vs.DeleteHandler)(w, r)
	case "PUT", "POST": // 写文件
		stats.WriteRequest()
		vs.guard.WhiteList(vs.PostHandler)(w, r)
	}
	stats.VolumeServerRequestHistogram.WithLabelValues(r.Method).Observe(time.Since(start).Seconds())
}
```

``` go

func (vs *VolumeServer) PostHandler(w http.ResponseWriter, r *http.Request) {
	if e := r.ParseForm(); e != nil {
		glog.V(0).Infoln("form parse error:", e)
		writeJsonError(w, r, http.StatusBadRequest, e)
		return
	}
	
	// url 的形式为: http://host:port/vid,fid (忽略 collection 等不重要的信息)
	vid, fid, _, _, _ := parseURLPath(r.URL.Path)
	// vid 为字符串， volumeId 为 uint32
	volumeId, _ := storage.NewVolumeId(vid)
	// jwt 校验
	vs.maybeCheckJwtAuthorization(r, vid, fid)
	
	// 从 http request 中读取一些文件的 meta
	needle, originalSize, _ := storage.CreateNeedleFromRequest(r, vs.FixJpgOrientation)

	ret := operation.UploadResult{}
	// 多副本写入数据
	_, _ = topology.ReplicatedWrite(vs.GetMaster(), vs.store, volumeId, needle, r)
	httpStatus := http.StatusCreated
	if needle.HasName() {
		ret.Name = string(needle.Name)
	}
	ret.Size = uint32(originalSize)
	ret.ETag = needle.Etag()
	setEtag(w, ret.ETag)
	writeJsonQuiet(w, r, httpStatus, ret)
}
```

``` go
func CreateNeedleFromRequest(r *http.Request, fixJpgOrientation bool) (n *Needle, originalSize int, e error) {
	var pairMap map[string]string
	fname, mimeType, isGzipped, isChunkedFile := "", "", false, false
	n = new(Needle)
	fname, n.Data, mimeType, pairMap, isGzipped, originalSize, n.LastModified, n.Ttl, isChunkedFile, e = ParseUpload(r)
	if len(fname) < 256 {
		n.Name = []byte(fname)
		n.SetHasName()
	}
	if len(mimeType) < 256 {
		n.Mime = []byte(mimeType)
		n.SetHasMime()
	}
	if len(pairMap) != 0 {
		trimmedPairMap := make(map[string]string)
		for k, v := range pairMap {
			trimmedPairMap[k[len(PairNamePrefix):]] = v
		}

		pairs, _ := json.Marshal(trimmedPairMap)
		if len(pairs) < 65536 {
			n.Pairs = pairs
			n.PairsSize = uint16(len(pairs))
			n.SetHasPairs()
		}
	}
	if isGzipped {
		n.SetGzipped()
	}
	if n.LastModified == 0 {
		n.LastModified = uint64(time.Now().Unix())
	}
	n.SetHasLastModifiedDate()
	if n.Ttl != EMPTY_TTL {
		n.SetHasTtl()
	}

	if isChunkedFile {
		n.SetIsChunkManifest()
	}

	if fixJpgOrientation {
		loweredName := strings.ToLower(fname)
		if mimeType == "image/jpeg" || strings.HasSuffix(loweredName, ".jpg") || strings.HasSuffix(loweredName, ".jpeg") {
			n.Data = images.FixJpgOrientation(n.Data)
		}
	}

	n.Checksum = NewCRC(n.Data)

	commaSep := strings.LastIndex(r.URL.Path, ",")
	dotSep := strings.LastIndex(r.URL.Path, ".")
	fid := r.URL.Path[commaSep+1:]
	if dotSep > 0 {
		fid = r.URL.Path[commaSep+1 : dotSep]
	}

	e = n.ParsePath(fid)

	return
}
```

``` go

func ReplicatedWrite(masterNode string, s *storage.Store,
	volumeId storage.VolumeId, needle *storage.Needle,
	r *http.Request) (size uint32, errorStatus string) {

	//check JWT
	jwt := security.GetJwt(r)
	// 执行完这个函数，数据就已经写入到本地 volume 中了
	ret, _ := s.Write(volumeId, needle)
	needToReplicate := !s.HasVolume(volumeId)


	needToReplicate = needToReplicate || s.GetVolume(volumeId).NeedToReplicate()
	if !needToReplicate {
		needToReplicate = s.GetVolume(volumeId).NeedToReplicate()
	}
	if needToReplicate { //send to other replica locations
		if r.FormValue("type") != "replicate" { // replica master 没有这个标记

			distributedOperation(masterNode, s, volumeId, func(location operation.Location) error {
				u := url.URL{
					Scheme: "http",
					Host:   location.Url,
					Path:   r.URL.Path,
				}
				q := url.Values{
					"type": {"replicate"}, // replica slaves 包含这个标记，避免重复写入
					"ttl":  {needle.Ttl.String()},
				}
				if needle.LastModified > 0 {
					q.Set("ts", strconv.FormatUint(needle.LastModified, 10))
				}
				if needle.IsChunkedManifest() {
					q.Set("cm", "true")
				}
				u.RawQuery = q.Encode()

				pairMap := make(map[string]string)
				if needle.HasPairs() {
					tmpMap := make(map[string]string)
					json.Unmarshal(needle.Pairs, &tmpMap)
					for k, v := range tmpMap {
						pairMap[storage.PairNamePrefix+k] = v
					}
				}

				// 向 replica 发送上传请求
				_, err := operation.Upload(u.String(),
					string(needle.Name), bytes.NewReader(needle.Data), needle.IsGzipped(), string(needle.Mime),
					pairMap, jwt)
				return err
			})
		}
	}
	size = ret
	return
}
```

``` go
func distributedOperation(masterNode string, store *storage.Store, volumeId storage.VolumeId, op func(location operation.Location) error) error {
	if lookupResult, lookupErr := operation.Lookup(masterNode, volumeId.String()); lookupErr == nil {
		// 从 master 读取这个 volume 的所有地址, 过滤本机，循环执行回调函数
		length := 0
		selfUrl := store.Ip + ":" + strconv.Itoa(store.Port)
		results := make(chan RemoteResult)
		for _, location := range lookupResult.Locations {
			if location.Url != selfUrl {
				length++
				// 每一个副本开启一个协程处理，并使用 channel 接收完成事件
				// 副本过多会对当前服务器出口带宽造成比较大的压力
				//
				// GFS 的方案: Figure 2
				//   https://static.googleusercontent.com/media/research.google.com/zh-CN//archive/gfs-sosp2003.pdf
				//
				go func(location operation.Location, results chan RemoteResult) {
					results <- RemoteResult{location.Url, op(location)}
				}(location, results)
			}
		}
		ret := DistributedOperationResult(make(map[string]error))
		for i := 0; i < length; i++ {
			result := <-results
			ret[result.Host] = result.Error
		}
		if volume := store.GetVolume(volumeId); volume != nil {
			if length+1 < volume.ReplicaPlacement.GetCopyCount() {
				return fmt.Errorf("replicating opetations [%d] is less than volume's replication copy count [%d]", length+1, volume.ReplicaPlacement.GetCopyCount())
			}
		}
		return ret.Error()
	} else {
		glog.V(0).Infoln()
		return fmt.Errorf("Failed to lookup for %d: %v", volumeId, lookupErr)
	}
}
```

### Read

``` go

func (vs *VolumeServer) GetOrHeadHandler(w http.ResponseWriter, r *http.Request) {
	n := new(storage.Needle)
	vid, fid, filename, ext, _ := parseURLPath(r.URL.Path)
	volumeId, err := storage.NewVolumeId(vid)
	if err != nil {
		glog.V(2).Infoln("parsing error:", err, r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = n.ParsePath(fid)
	if err != nil {
		glog.V(2).Infoln("parsing fid error:", err, r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	glog.V(4).Infoln("volume", volumeId, "reading", n)
	if !vs.store.HasVolume(volumeId) {
		// 如果目标 volume 不在当前服务，根据是否启用重定向读
		// 启用:
		//     去 master 查询目标 volume 所在位置，重定向请求
		//
		// 未启用:
		//     返回 404
		if !vs.ReadRedirect {
			glog.V(2).Infoln("volume is not local:", err, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		lookupResult, err := operation.Lookup(vs.GetMaster(), volumeId.String())
		glog.V(2).Infoln("volume", volumeId, "found on", lookupResult, "error", err)
		if err == nil && len(lookupResult.Locations) > 0 {
			u, _ := url.Parse(util.NormalizeUrl(lookupResult.Locations[0].PublicUrl))
			u.Path = r.URL.Path
			arg := url.Values{}
			if c := r.FormValue("collection"); c != "" {
				arg.Set("collection", c)
			}
			u.RawQuery = arg.Encode()
			//
			http.Redirect(w, r, u.String(), http.StatusMovedPermanently)

		} else {
			glog.V(2).Infoln("lookup error:", err, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}
	// 读取数据并校验
	cookie := n.Cookie
	count, e := vs.store.ReadVolumeNeedle(volumeId, n)
	glog.V(4).Infoln("read bytes", count, "error", e)
	if e != nil || count < 0 {
		glog.V(0).Infof("read %s error: %v", r.URL.Path, e)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if n.Cookie != cookie {
		glog.V(0).Infof("request %s with cookie:%x expected:%x from %s agent %s", r.URL.Path, cookie, n.Cookie, r.RemoteAddr, r.UserAgent())
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if n.LastModified != 0 {
		w.Header().Set("Last-Modified", time.Unix(int64(n.LastModified), 0).UTC().Format(http.TimeFormat))
		if r.Header.Get("If-Modified-Since") != "" {
			if t, parseError := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since")); parseError == nil {
				if t.Unix() >= int64(n.LastModified) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
		}
	}
	if inm := r.Header.Get("If-None-Match"); inm == "\""+n.Etag()+"\"" {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if r.Header.Get("ETag-MD5") == "True" {
		setEtag(w, n.MD5())
	} else {
		setEtag(w, n.Etag())
	}

	if n.HasPairs() {
		pairMap := make(map[string]string)
		err = json.Unmarshal(n.Pairs, &pairMap)
		if err != nil {
			glog.V(0).Infoln("Unmarshal pairs error:", err)
		}
		for k, v := range pairMap {
			w.Header().Set(k, v)
		}
	}

	if vs.tryHandleChunkedFile(n, filename, w, r) {
		return
	}

	if n.NameSize > 0 && filename == "" {
		filename = string(n.Name)
		if ext == "" {
			ext = path.Ext(filename)
		}
	}
	mtype := ""
	if n.MimeSize > 0 {
		mt := string(n.Mime)
		if !strings.HasPrefix(mt, "application/octet-stream") {
			mtype = mt
		}
	}

	if ext != ".gz" {
		if n.IsGzipped() {
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				w.Header().Set("Content-Encoding", "gzip")
			} else {
				if n.Data, err = operation.UnGzipData(n.Data); err != nil {
					glog.V(0).Infoln("ungzip error:", err, r.URL.Path)
				}
			}
		}
	}

	rs := conditionallyResizeImages(bytes.NewReader(n.Data), ext, r)

	if e := writeResponseContent(filename, mtype, rs, w, r); e != nil {
		glog.V(2).Infoln("response write error:", e)
	}
}
```

``` go
func (s *Store) ReadVolumeNeedle(i VolumeId, n *Needle) (int, error) {
	if v := s.findVolume(i); v != nil {
		return v.readNeedle(n)
	}
	return 0, fmt.Errorf("Volume %d not found!", i)
}


func (s *Store) findVolume(vid VolumeId) *Volume {
	for _, location := range s.Locations {
		if v, found := location.FindVolume(vid); found {
			return v
		}
	}
	return nil
}


// read fills in Needle content by looking up n.Id from NeedleMapper
func (v *Volume) readNeedle(n *Needle) (int, error) {
	nv, ok := v.nm.Get(n.Id)
	if !ok || nv.Offset.IsZero() {
		v.compactingWg.Wait()
		nv, ok = v.nm.Get(n.Id)
		if !ok || nv.Offset.IsZero() {
			return -1, ErrorNotFound
		}
	}
	if nv.Size == TombstoneFileSize {
		return -1, errors.New("already deleted")
	}
	if nv.Size == 0 {
		return 0, nil
	}
	err := n.ReadData(v.dataFile, nv.Offset.ToAcutalOffset(), nv.Size, v.Version())
	if err != nil {
		return 0, err
	}
	bytesRead := len(n.Data)
	if !n.HasTtl() {
		return bytesRead, nil
	}
	ttlMinutes := n.Ttl.Minutes()
	if ttlMinutes == 0 {
		return bytesRead, nil
	}
	if !n.HasLastModifiedDate() {
		return bytesRead, nil
	}
	if uint64(time.Now().Unix()) < n.LastModified+uint64(ttlMinutes*60) {
		return bytesRead, nil
	}
	return -1, ErrorNotFound
}

```

### Delete

``` go
func (v *Volume) deleteNeedle(n *Needle) (uint32, error) {
	if v.readOnly {
		return 0, fmt.Errorf("%s is read-only", v.dataFile.Name())
	}
	v.dataFileAccessLock.Lock()
	defer v.dataFileAccessLock.Unlock()
	// 使用 needle id 查找索引
	nv, ok := v.nm.Get(n.Id)
	// 存在 needle 并且未被删除 (将 size 设置为 TombstoneFileSize 表示删除)
	if ok && nv.Size != TombstoneFileSize {
		size := nv.Size // size 作为返回值
		n.Data = nil // 将数据设置为空
		n.AppendAtNs = uint64(time.Now().UnixNano()) // 更新时间
		// !!! 然后，把这个新的 needle 写入文件
		offset, _, _, _ := n.Append(v.dataFile, v.Version())
		// 从索引中删除
		v.nm.Delete(n.Id, ToOffset(int64(offset)))
		return size, nil
	}
	return 0, nil
}

func (nm *NeedleMap) Delete(key types.NeedleId, offset types.Offset) error {
	deletedBytes := nm.m.Delete(types.NeedleId(key))
	nm.logDelete(deletedBytes)
	return nm.appendToIndexFile(key, offset, types.TombstoneFileSize)
}
```
