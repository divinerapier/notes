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

在这个函数内包含了，读取配置，生成启动参数，启动 `http/https` 及 `grpc` 服务。`http` 服务的 `API` 由 `adminMux` 及 `publicMux` 组成，在函数 `NewVolumeServer` 中定义。

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



