# Vacuum

手工执行一次 `GC` 需要由 `master` & `volume` 协同完成。用户通过访问 `master` 接口申请执行一次 `GC` 操作：

`curl http://master:9333/vol/vacuum?garbageThreshold=0.001` 是 `seaweedfs` 的垃圾回收接口。

之后，`master` 会通过 `grpc` 调用 `volume` 的接口控制 `GC` 流程。

由 `volume server` 的4个接口共同提供服务。

``` go
type VolumeServerServer interface {
    VacuumVolumeCheck(VacuumVolumeCheckRequest) VacuumVolumeCheckResponse
    VacuumVolumeCompact(VacuumVolumeCompactRequest) VacuumVolumeCompactResponse
    VacuumVolumeCommit(VacuumVolumeCommitRequest) VacuumVolumeCommitResponse
    VacuumVolumeCleanup(VacuumVolumeCleanupRequest) VacuumVolumeCleanupResponse
}
```



## Master API

手工触发 `GC`

``` go
func (ms *MasterServer) volumeVacuumHandler(w http.ResponseWriter, r *http.Request) {
	gcString := r.FormValue("garbageThreshold")
	gcThreshold := ms.garbageThreshold
	if gcString != "" {
		gcThreshold, _ = strconv.ParseFloat(gcString, 32)
	}
	ms.Topo.Vacuum(ms.grpcDialOpiton, gcThreshold, ms.preallocate)
	ms.dirStatusHandler(w, r)
}

func (t *Topology) Vacuum(grpcDialOption grpc.DialOption, garbageThreshold float64, preallocate int64) int {
	for _, col := range t.collectionMap.Items() {
		c := col.(*Collection)
		for _, vl := range c.storageType2VolumeLayout.Items() {
			if vl != nil {
				volumeLayout := vl.(*VolumeLayout)
				vacuumOneVolumeLayout(grpcDialOption, volumeLayout, c, garbageThreshold, preallocate)
			}
		}
	}
	return 0
}


func vacuumOneVolumeLayout(grpcDialOption grpc.DialOption, volumeLayout *VolumeLayout, c *Collection, garbageThreshold float64, preallocate int64) {

	volumeLayout.accessLock.RLock()
	tmpMap := make(map[storage.VolumeId]*VolumeLocationList)
	for vid, locationList := range volumeLayout.vid2location {
		tmpMap[vid] = locationList
	}
	volumeLayout.accessLock.RUnlock()

	for vid, locationList := range tmpMap {
		// 蜜汁枷锁
		volumeLayout.accessLock.RLock()
		isReadOnly, hasValue := volumeLayout.readonlyVolumes[vid]
		volumeLayout.accessLock.RUnlock()

		// 跳过只读 volume
		if hasValue && isReadOnly {
			continue
		}

		// 访问 volume service 询问确认是否需要执行 GC, 无效数据超过设置阈值时需要执行
		if batchVacuumVolumeCheck(grpcDialOption, volumeLayout, vid, locationList, garbageThreshold) {
			// 执行 GC
			if batchVacuumVolumeCompact(grpcDialOption, volumeLayout, vid, locationList, preallocate) {
				// 提交 GC
				batchVacuumVolumeCommit(grpcDialOption, volumeLayout, vid, locationList)
			} else {
				// 回滚
				batchVacuumVolumeCleanup(grpcDialOption, volumeLayout, vid, locationList)
			}
		}
	}
}


func batchVacuumVolumeCheck(grpcDialOption grpc.DialOption, vl *VolumeLayout, vid storage.VolumeId, locationlist *VolumeLocationList, garbageThreshold float64) bool {
	ch := make(chan bool, locationlist.Length())
	for index, dn := range locationlist.list {
		go func(index int, url string, vid storage.VolumeId) {
			err := operation.WithVolumeServerClient(url, grpcDialOption, func(volumeServerClient volume_server_pb.VolumeServerClient) error {
				resp, err := volumeServerClient.VacuumVolumeCheck(context.Background(), &volume_server_pb.VacuumVolumeCheckRequest{
					VolumeId: uint32(vid),
				})
				if err != nil {
					ch <- false
					return err
				}
				isNeeded := resp.GarbageRatio > garbageThreshold
				ch <- isNeeded
				return nil
			})
			if err != nil {
				glog.V(0).Infof("Checking vacuuming %d on %s: %v", vid, url, err)
			}
		}(index, dn.Url(), vid)
	}
	isCheckSuccess := true
	for range locationlist.list {
		select {
		case canVacuum := <-ch:
			isCheckSuccess = isCheckSuccess && canVacuum
		case <-time.After(30 * time.Minute):
			isCheckSuccess = false
			break
		}
	}
	return isCheckSuccess
}
func batchVacuumVolumeCompact(grpcDialOption grpc.DialOption, vl *VolumeLayout, vid storage.VolumeId, locationlist *VolumeLocationList, preallocate int64) bool {
	vl.removeFromWritable(vid)
	ch := make(chan bool, locationlist.Length())
	for index, dn := range locationlist.list {
		go func(index int, url string, vid storage.VolumeId) {
			glog.V(0).Infoln(index, "Start vacuuming", vid, "on", url)
			err := operation.WithVolumeServerClient(url, grpcDialOption, func(volumeServerClient volume_server_pb.VolumeServerClient) error {
				_, err := volumeServerClient.VacuumVolumeCompact(context.Background(), &volume_server_pb.VacuumVolumeCompactRequest{
					VolumeId: uint32(vid),
				})
				return err
			})
			if err != nil {
				glog.Errorf("Error when vacuuming %d on %s: %v", vid, url, err)
				ch <- false
			} else {
				glog.V(0).Infof("Complete vacuuming %d on %s", vid, url)
				ch <- true
			}
		}(index, dn.Url(), vid)
	}
	isVacuumSuccess := true
	for range locationlist.list {
		select {
		case canCommit := <-ch:
			isVacuumSuccess = isVacuumSuccess && canCommit
		case <-time.After(30 * time.Minute):
			isVacuumSuccess = false
			break
		}
	}
	return isVacuumSuccess
}
func batchVacuumVolumeCommit(grpcDialOption grpc.DialOption, vl *VolumeLayout, vid storage.VolumeId, locationlist *VolumeLocationList) bool {
	isCommitSuccess := true
	for _, dn := range locationlist.list {
		glog.V(0).Infoln("Start Committing vacuum", vid, "on", dn.Url())
		err := operation.WithVolumeServerClient(dn.Url(), grpcDialOption, func(volumeServerClient volume_server_pb.VolumeServerClient) error {
			_, err := volumeServerClient.VacuumVolumeCommit(context.Background(), &volume_server_pb.VacuumVolumeCommitRequest{
				VolumeId: uint32(vid),
			})
			return err
		})
		if err != nil {
			glog.Errorf("Error when committing vacuum %d on %s: %v", vid, dn.Url(), err)
			isCommitSuccess = false
		} else {
			glog.V(0).Infof("Complete Committing vacuum %d on %s", vid, dn.Url())
		}
		if isCommitSuccess {
			vl.SetVolumeAvailable(dn, vid)
		}
	}
	return isCommitSuccess
}
func batchVacuumVolumeCleanup(grpcDialOption grpc.DialOption, vl *VolumeLayout, vid storage.VolumeId, locationlist *VolumeLocationList) {
	for _, dn := range locationlist.list {
		glog.V(0).Infoln("Start cleaning up", vid, "on", dn.Url())
		err := operation.WithVolumeServerClient(dn.Url(), grpcDialOption, func(volumeServerClient volume_server_pb.VolumeServerClient) error {
			_, err := volumeServerClient.VacuumVolumeCleanup(context.Background(), &volume_server_pb.VacuumVolumeCleanupRequest{
				VolumeId: uint32(vid),
			})
			return err
		})
		if err != nil {
			glog.Errorf("Error when cleaning up vacuum %d on %s: %v", vid, dn.Url(), err)
		} else {
			glog.V(0).Infof("Complete cleaning up vacuum %d on %s", vid, dn.Url())
		}
	}
}

```
