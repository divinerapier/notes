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



```
