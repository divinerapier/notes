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

	ms.guard = security.NewGuard(whiteList, signingKey)

	if !disableHttp {
		handleStaticResources2(r)
		r.HandleFunc("/", ms.proxyToLeader(ms.uiStatusHandler))
		r.HandleFunc("/ui/index.html", ms.uiStatusHandler)
		
		// 将请求转发给 master
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
	
	// 刷新 volume 状态后台线程
	ms.Topo.StartRefreshWritableVolumes(ms.grpcDialOpiton, garbageThreshold, ms.preallocate)

	return ms
}
```

## HTTP API

### dirAssignHandler

``` go

func (ms *MasterServer) dirAssignHandler(w http.ResponseWriter, r *http.Request) {
	//stats.AssignRequest()
	requestedCount, e := strconv.ParseUint(r.FormValue("count"), 10, 64)
	if e != nil || requestedCount == 0 {
		requestedCount = 1
	}

	option, _ := ms.getVolumeGrowOption(r)

	if !ms.Topo.HasWritableVolume(option) {
		if ms.Topo.FreeSpace() <= 0 {
			writeJsonQuiet(w, r, http.StatusNotFound, operation.AssignResult{Error: "No free volumes left!"})
			return
		}
		ms.vgLock.Lock()
		defer ms.vgLock.Unlock()
		if !ms.Topo.HasWritableVolume(option) {
			// 申请创建新的 volume
			if _, err = ms.vg.AutomaticGrowByType(option, ms.grpcDialOpiton, ms.Topo); err != nil {
				writeJsonError(w, r, http.StatusInternalServerError,
					fmt.Errorf("Cannot grow volume group! %v", err))
				return
			}
		}
	}
	fid, count, dn, err := ms.Topo.PickForWrite(requestedCount, option)
	if err == nil {
		ms.maybeAddJwtAuthorization(w, fid)
		writeJsonQuiet(w, r, http.StatusOK, operation.AssignResult{Fid: fid, Url: dn.Url(), PublicUrl: dn.PublicUrl, Count: count})
	} else {
		writeJsonQuiet(w, r, http.StatusNotAcceptable, operation.AssignResult{Error: err.Error()})
	}
}

func (vg *VolumeGrowth) AutomaticGrowByType(option *VolumeGrowOption, grpcDialOption grpc.DialOption, topo *Topology) (count int, err error) {
	count, err = vg.GrowByCountAndType(grpcDialOption, vg.findVolumeCount(option.ReplicaPlacement.GetCopyCount()), option, topo)
	if count > 0 && count%option.ReplicaPlacement.GetCopyCount() == 0 {
		return count, nil
	}
	return count, err
}

func (vg *VolumeGrowth) GrowByCountAndType(grpcDialOption grpc.DialOption, targetCount int, option *VolumeGrowOption, topo *Topology) (counter int, err error) {
	vg.accessLock.Lock()
	defer vg.accessLock.Unlock()

	for i := 0; i < targetCount; i++ {
		if c, e := vg.findAndGrow(grpcDialOption, topo, option); e == nil {
			counter += c
		} else {
			return counter, e
		}
	}
	return
}

func (vg *VolumeGrowth) findAndGrow(grpcDialOption grpc.DialOption, topo *Topology, option *VolumeGrowOption) (int, error) {
	// 找到满足需求的
	servers, e := vg.findEmptySlotsForOneVolume(topo, option)
	if e != nil {
		return 0, e
	}
	vid, raftErr := topo.NextVolumeId()
	if raftErr != nil {
		return 0, raftErr
	}
	err := vg.grow(grpcDialOption, topo, vid, option, servers...)
	return len(servers), err
}

// 遍历所有的 Node，申请使用指定 vid 创建一个 volume (创建逻辑在 volume 篇)
func (vg *VolumeGrowth) grow(grpcDialOption grpc.DialOption, topo *Topology, vid storage.VolumeId, option *VolumeGrowOption, servers ...*DataNode) error {
	for _, server := range servers {
		if err := AllocateVolume(server, grpcDialOption, vid, option); err == nil {
			vi := storage.VolumeInfo{
				Id:               vid,
				Size:             0,
				Collection:       option.Collection,
				ReplicaPlacement: option.ReplicaPlacement,
				Ttl:              option.Ttl,
				Version:          storage.CurrentVersion,
			}
			server.AddOrUpdateVolume(vi)
			topo.RegisterVolumeLayout(vi, server)
			glog.V(0).Infoln("Created Volume", vid, "on", server.NodeImpl.String())
		} else {
			glog.V(0).Infoln("Failed to assign volume", vid, "to", servers, "error", err)
			return fmt.Errorf("Failed to assign %d: %v", vid, err)
		}
	}
	return nil
}
```
