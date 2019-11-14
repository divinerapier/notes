# Master Service

## Start Service

`master service` 起始于 `weed/command/master.go`

``` go

func runMaster(cmd *Command, args []string) bool {

	weed_server.LoadConfiguration("security", false)

	if *mMaxCpu < 1 {
		*mMaxCpu = runtime.NumCPU()
	}
	runtime.GOMAXPROCS(*mMaxCpu)
	util.SetupProfiling(*masterCpuProfile, *masterMemProfile)

	if err := util.TestFolderWritable(*metaFolder); err != nil {
		glog.Fatalf("Check Meta Folder (-mdir) Writable %s : %s", *metaFolder, err)
	}
	if *masterWhiteListOption != "" {
		masterWhiteList = strings.Split(*masterWhiteListOption, ",")
	}
  
  // 标准版本中， volume 最大为 30G
	if *volumeSizeLimitMB > util.VolumeSizeLimitGB*1000 {
		glog.Fatalf("volumeSizeLimitMB should be smaller than 30000")
	}

	metrics.MasterRegisterMetrics()
	r := mux.NewRouter()
  
  // !!!
	ms := weed_server.NewMasterServer(r, *mport, *metaFolder,
		*volumeSizeLimitMB, *volumePreallocate,
		*mpulse, *defaultReplicaPlacement, *garbageThreshold,
		masterWhiteList,
		*disableHttp,
		*metricsAddress,     // prometheus 地址
		*metricsIntervalSec, // 在master中设置监控时间间隔，通过心跳的方式
	)

	listeningAddress := *masterBindIp + ":" + strconv.Itoa(*mport)

	glog.V(0).Infoln("Start Seaweed Master", util.VERSION, "at", listeningAddress)

	masterListener, _ := util.NewListener(listeningAddress, 0)

	go func() {
		// start raftServer
		myMasterAddress, peers := checkPeers(*masterIp, *mport, *masterPeers)
    
    // 启动 Raft Server
		raftServer := weed_server.NewRaftServer(security.LoadClientTLS(viper.Sub("grpc"), "master"),
			peers, myMasterAddress, *metaFolder, ms.Topo, *mpulse)
		ms.SetRaftServer(raftServer)
		r.HandleFunc("/cluster/status", raftServer.StatusHandler).Methods("GET")

		// starting grpc server
		grpcPort := *mport + 10000
		grpcL, _ := util.NewListener(*masterBindIp+":"+strconv.Itoa(grpcPort), 0)
		// Create your protocol servers.
		grpcS := util.NewGrpcServer(security.LoadServerTLS(viper.Sub("grpc"), "master"))
		master_pb.RegisterSeaweedServer(grpcS, ms)
		protobuf.RegisterRaftServer(grpcS, raftServer)
		reflection.Register(grpcS)
		grpcS.Serve(grpcL)
	}()

	// start http server
	httpS := &http.Server{Handler: r}
  httpS.Serve(masterListener)

	return true
}
```

``` go

func NewMasterServer(r *mux.Router, port int, metaFolder string,
	volumeSizeLimitMB uint,
	preallocate bool,
	pulseSeconds int, // 刷新可写 volume 时间间隔, (*Topology).StartRefreshWritableVolumes 使用
	defaultReplicaPlacement string,
	garbageThreshold float64, // gc 阈值，垃圾数据占比超过时执行 gc
	whiteList []string,
	disableHttp bool,
	metricsAddress string,
	metricsIntervalSec int,
) *MasterServer {

	v := viper.GetViper()
	signingKey := v.GetString("jwt.signing.key")

	var preallocateSize int64
	if preallocate {
		preallocateSize = int64(volumeSizeLimitMB) * (1 << 20)
	}
	ms := &MasterServer{
		port:                    port,
		volumeSizeLimitMB:       volumeSizeLimitMB,
		preallocate:             preallocateSize,
		pulseSeconds:            pulseSeconds,
		defaultReplicaPlacement: defaultReplicaPlacement,
		garbageThreshold:        garbageThreshold,
		clientChans:             make(map[string]chan *master_pb.VolumeLocation),
		grpcDialOpiton:          security.LoadClientTLS(v.Sub("grpc"), "master"),
		metricsAddress:          metricsAddress,
		metricsIntervalSec:      metricsIntervalSec,
	}
	ms.bounedLeaderChan = make(chan int, 16)
	seq := sequence.NewMemorySequencer()
	ms.Topo = topology.NewTopology("topo", seq, uint64(volumeSizeLimitMB)*1024*1024, pulseSeconds)
	ms.vg = topology.NewDefaultVolumeGrowth()
	glog.V(0).Infoln("Volume Size Limit is", volumeSizeLimitMB, "MB")

	ms.guard = security.NewGuard(whiteList, signingKey)

	if !disableHttp {
		handleStaticResources2(r)
		r.HandleFunc("/", ms.proxyToLeader(ms.uiStatusHandler))
		r.HandleFunc("/ui/index.html", ms.uiStatusHandler)
		r.HandleFunc("/dir/assign", ms.proxyToLeader(ms.guard.WhiteList(ms.dirAssignHandler)))
		r.HandleFunc("/dir/lookup", ms.proxyToLeader(ms.guard.WhiteList(ms.dirLookupHandler)))
		r.HandleFunc("/dir/status", ms.proxyToLeader(ms.guard.WhiteList(ms.dirStatusHandler)))
		r.HandleFunc("/col/delete", ms.proxyToLeader(ms.guard.WhiteList(ms.collectionDeleteHandler)))
		r.HandleFunc("/vol/grow", ms.proxyToLeader(ms.guard.WhiteList(ms.volumeGrowHandler)))
		r.HandleFunc("/vol/status", ms.proxyToLeader(ms.guard.WhiteList(ms.volumeStatusHandler)))
		r.HandleFunc("/vol/vacuum", ms.proxyToLeader(ms.guard.WhiteList(ms.volumeVacuumHandler)))
		r.HandleFunc("/submit", ms.guard.WhiteList(ms.submitFromMasterServerHandler))
		r.HandleFunc("/healthz", ms.guard.WhiteList(healthzHandler))
		r.HandleFunc("/metrics", ms.guard.WhiteList(ms.metricsHandler))
		r.HandleFunc("/{fileId}", ms.proxyToLeader(ms.redirectHandler))
	}

	ms.Topo.StartRefreshWritableVolumes(ms.grpcDialOpiton, garbageThreshold, ms.preallocate)

	return ms
}
```
