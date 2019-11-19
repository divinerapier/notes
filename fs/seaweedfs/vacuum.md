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
	// 分层次: collection -> volume 遍历 (Items 返回一个快照)
	// type VolumeLayout struct {
	// 	rp               *storage.ReplicaPlacement
	// 	ttl              *storage.TTL
	// 	vid2location     map[storage.VolumeId]*VolumeLocationList
	// 	writables        []storage.VolumeId        // transient array of writable volume id
	// 	readonlyVolumes  map[storage.VolumeId]bool // transient set of readonly volumes
	// 	oversizedVolumes map[storage.VolumeId]bool // set of oversized volumes
	// 	volumeSizeLimit  uint64
	// 	accessLock       sync.RWMutex
	// }
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

以下四个函数为 `GC` 的四个步骤。在结构上是保持一致的。

``` go

func batchVacuumVolumeCheck(
	grpcDialOption grpc.DialOption, 
	vl *VolumeLayout,			
	vid storage.VolumeId, 			// 目标 volume
	locationlist *VolumeLocationList, 	// 包含 volume 所有节点的地址 
	garbageThreshold float64,		// 用户输入的阈值，默认为 0.3
) bool {
	
	ch := make(chan bool, locationlist.Length())
	for index, dn := range locationlist.list {
		go func(index int, url string, vid storage.VolumeId) {
			// 通过 grpc 访问 volume server
			err := operation.WithVolumeServerClient(url, grpcDialOption, func(volumeServerClient volume_server_pb.VolumeServerClient) error {
				resp, err := volumeServerClient.VacuumVolumeCheck(context.Background(), &volume_server_pb.VacuumVolumeCheckRequest{
					VolumeId: uint32(vid),
				})
				if err != nil {
					ch <- false
					return err
				}
				// 垃圾比例大于阈值
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
			// 应该有两个问题:
			// 1. 应该在外部创建 after := time.After(30 * time.Minute), 否则会每一次都创建一个定时器
			// 2. break 无法跳出 for { select {} } 块
			isCheckSuccess = false
			break
		}
	}
	return isCheckSuccess
}

func batchVacuumVolumeCompact(grpcDialOption grpc.DialOption, vl *VolumeLayout, vid storage.VolumeId, locationlist *VolumeLocationList, preallocate int64) bool {
	// 标记目标 volume 为不可写入状态
	vl.removeFromWritable(vid)
	ch := make(chan bool, locationlist.Length())
	for index, dn := range locationlist.list {
		go func(index int, url string, vid storage.VolumeId) {
			operation.WithVolumeServerClient(url, grpcDialOption, func(volumeServerClient volume_server_pb.VolumeServerClient) error {
				_, _ = volumeServerClient.VacuumVolumeCompact(context.Background(), &volume_server_pb.VacuumVolumeCompactRequest{
					VolumeId: uint32(vid),
				})
				return nil
			})
			if err != nil {
				ch <- false
			} else {
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
		err := operation.WithVolumeServerClient(dn.Url(), grpcDialOption, func(volumeServerClient volume_server_pb.VolumeServerClient) error {
			_, err := volumeServerClient.VacuumVolumeCommit(context.Background(), &volume_server_pb.VacuumVolumeCommitRequest{
				VolumeId: uint32(vid),
			})
			return err
		})
		isCommitSuccess = err == nil
		if isCommitSuccess {
			vl.SetVolumeAvailable(dn, vid)
		}
	}
	return isCommitSuccess
}

func batchVacuumVolumeCleanup(grpcDialOption grpc.DialOption, vl *VolumeLayout, vid storage.VolumeId, locationlist *VolumeLocationList) {
	for _, dn := range locationlist.list {
		operation.WithVolumeServerClient(dn.Url(), grpcDialOption, func(volumeServerClient volume_server_pb.VolumeServerClient) error {
			_, _ = volumeServerClient.VacuumVolumeCleanup(context.Background(), &volume_server_pb.VacuumVolumeCleanupRequest{
				VolumeId: uint32(vid),
			})
			return nil
		})
	}
}

```

## Volume GRPC

### VacuumVolumeCheck

``` go

func (vs *VolumeServer) VacuumVolumeCheck(ctx context.Context, req *volume_server_pb.VacuumVolumeCheckRequest) (*volume_server_pb.VacuumVolumeCheckResponse, error) {

	resp := &volume_server_pb.VacuumVolumeCheckResponse{}

	garbageRatio, err := vs.store.CheckCompactVolume(storage.VolumeId(req.VolumeId))

	resp.GarbageRatio = garbageRatio

	return resp, err

}

func (s *Store) CheckCompactVolume(volumeId VolumeId) (float64, error) {
	if v := s.findVolume(volumeId); v != nil {
		// 直接读取内存中的数据，在删除覆盖的时候立即更新
		return v.garbageLevel(), nil
	}
	return 0, fmt.Errorf("volume id %d is not found during check compact", volumeId)
}

```

### VacuumVolumeCompact

``` go

func (vs *VolumeServer) VacuumVolumeCompact(ctx context.Context, req *volume_server_pb.VacuumVolumeCompactRequest) (*volume_server_pb.VacuumVolumeCompactResponse, error) {
	resp := &volume_server_pb.VacuumVolumeCompactResponse{}
	err := vs.store.CompactVolume(storage.VolumeId(req.VolumeId), req.Preallocate)
	return resp, err
}

func (s *Store) CompactVolume(vid VolumeId, preallocate int64) error {
	v := s.findVolume(vid)
	return v.Compact(preallocate)
}

func (v *Volume) Compact(preallocate int64) error {
	filePath := v.FileName() // path/to/dir/(collection_)id
	v.lastCompactIndexOffset = v.nm.IndexFileSize()
	v.lastCompactRevision = v.SuperBlock.CompactRevision
	return v.copyDataAndGenerateIndexFile(filePath+".cpd", filePath+".cpx", preallocate)
}


func (v *Volume) copyDataAndGenerateIndexFile(dstName, idxName string, preallocate int64) (err error) {
	// dstName: path/to/dir/(collection_)id.cpd
	// idxName: path/to/dir/(collection_)id.cpx
	// 创建文件，没有多余操作 preallocate 是无效参数
	dst, _ := createVolumeFile(dstName, preallocate)
	defer dst.Close()
	idx, _ := os.OpenFile(idxName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	defer idx.Close()

	scanner := &VolumeFileScanner4Vacuum{
		v:   v,				// 原有 volume 对象
		now: uint64(time.Now().Unix()),
		nm:  NewBtreeNeedleMap(idx), 	// 新的索引文件
		dst: dst,
	}
	err = ScanVolumeFile(v.dir, v.Collection, v.Id, v.needleMapKind, scanner)
	return
}


func ScanVolumeFile(dirname string, collection string, id VolumeId,
	needleMapKind NeedleMapType,
	volumeFileScanner VolumeFileScanner) (err error) {

	// 从磁盘(根据dirname, collection, id) 读取 volume data 文件 (不加载索引文件)
	v, _ := loadVolumeWithoutIndex(dirname, collection, id, needleMapKind)
	
	// func (scanner *VolumeFileScanner4Vacuum) VisitSuperBlock(superBlock SuperBlock) error {
	// 	scanner.version = superBlock.Version()
	//	superBlock.CompactRevision++
	//	_, err := scanner.dst.Write(superBlock.Bytes())
	//	scanner.newOffset = int64(superBlock.BlockSize())
	//	return err
	// }

	// 从现有的 volume 数据文件中读取 super block，更新 compact revision 写入到新的文件中
	volumeFileScanner.VisitSuperBlock(v.SuperBlock)
	defer v.Close()

	version := v.Version()
	offset := int64(v.SuperBlock.BlockSize())
	
	// version: 数据格式版本
	// v.dataFile: 磁盘上的数据文件
	// offset: 新创建的 volume 对象记录的偏移量 (应该是 8Bytes)
	// volumeFileScanner: 扫描方式
	return ScanVolumeFileFrom(version, v.dataFile, offset, volumeFileScanner)
}


func ScanVolumeFileFrom(version Version, dataFile *os.File, offset int64, volumeFileScanner VolumeFileScanner) (err error) {
	n, rest, _ := ReadNeedleHeader(dataFile, version, offset)
	
	// 迭代从
	for n != nil {
		if volumeFileScanner.ReadNeedleBody() { // always returns true
			// 读取 needle
			n.ReadNeedleBody(dataFile, version, offset+NeedleEntrySize, rest)
		}
		err := volumeFileScanner.VisitNeedle(n, offset)
		if err == io.EOF {
			return nil
		}
		offset += NeedleEntrySize + rest
		n, rest, _ = ReadNeedleHeader(dataFile, version, offset)
	}
	return nil
}

```

#### Volume File Scanner For Vacuum

``` go

type VolumeFileScanner4Vacuum struct {
	version   Version
	v         *Volume
	dst       *os.File
	nm        *NeedleMap
	newOffset int64
	now       uint64
}

func (scanner *VolumeFileScanner4Vacuum) VisitSuperBlock(superBlock SuperBlock) error {
	scanner.version = superBlock.Version()
	superBlock.CompactRevision++
	_, err := scanner.dst.Write(superBlock.Bytes())
	scanner.newOffset = int64(superBlock.BlockSize())
	return err

}
func (scanner *VolumeFileScanner4Vacuum) ReadNeedleBody() bool {
	return true
}

func (scanner *VolumeFileScanner4Vacuum) VisitNeedle(n *Needle, offset int64) error {
	// 过滤过期数据
	if n.HasTtl() && scanner.now >= n.LastModified+uint64(scanner.v.Ttl.Minutes()*60) {
		return nil
	}
	nv, ok := scanner.v.nm.Get(n.Id)
	// 1. 在索引中能找到
	// 2. 索引中的偏移量与传入的偏移量相同(因为写入同名 needle 会更新偏移量)
	// 3. 大小合法
	if ok && nv.Offset.ToAcutalOffset() == offset && nv.Size > 0 && nv.Size != TombstoneFileSize {
		scanner.nm.Put(n.Id, ToOffset(scanner.newOffset), n.Size)
		n.Append(scanner.dst, scanner.v.Version())
		scanner.newOffset += n.DiskSize(scanner.version)
	}
	return nil
}
```

### VacuumVolumeCommit

``` go

func (vs *VolumeServer) VacuumVolumeCommit(ctx context.Context, req *volume_server_pb.VacuumVolumeCommitRequest) (*volume_server_pb.VacuumVolumeCommitResponse, error) {

	resp := &volume_server_pb.VacuumVolumeCommitResponse{}

	err := vs.store.CommitCompactVolume(storage.VolumeId(req.VolumeId))

	if err != nil {
		glog.Errorf("commit volume %d: %v", req.VolumeId, err)
	} else {
		glog.V(1).Infof("commit volume %d", req.VolumeId)
	}

	return resp, err

}

func (s *Store) CommitCompactVolume(vid VolumeId) error {
	if v := s.findVolume(vid); v != nil {
		return v.CommitCompact()
	}
	return fmt.Errorf("volume id %d is not found during commit compact", vid)
}


func (v *Volume) CommitCompact() error {
	glog.V(0).Infof("Committing volume %d vacuuming...", v.Id)
	v.dataFileAccessLock.Lock()
	defer v.dataFileAccessLock.Unlock()
	glog.V(3).Infof("Got volume %d committing lock...", v.Id)
	v.compactingWg.Add(1)
	defer v.compactingWg.Done()
	v.nm.Close()
	if err := v.dataFile.Close(); err != nil {
		glog.V(0).Infof("fail to close volume %d", v.Id)
	}
	v.dataFile = nil

	var e error
	if e = v.makeupDiff(v.FileName()+".cpd", v.FileName()+".cpx", v.FileName()+".dat", v.FileName()+".idx"); e != nil {
		glog.V(0).Infof("makeupDiff in CommitCompact volume %d failed %v", v.Id, e)
		e = os.Remove(v.FileName() + ".cpd")
		if e != nil {
			return e
		}
		e = os.Remove(v.FileName() + ".cpx")
		if e != nil {
			return e
		}
	} else {
		var e error
		if e = os.Rename(v.FileName()+".cpd", v.FileName()+".dat"); e != nil {
			return fmt.Errorf("rename %s: %v", v.FileName()+".cpd", e)
		}
		if e = os.Rename(v.FileName()+".cpx", v.FileName()+".idx"); e != nil {
			return fmt.Errorf("rename %s: %v", v.FileName()+".cpx", e)
		}
	}

	//glog.V(3).Infof("Pretending to be vacuuming...")
	//time.Sleep(20 * time.Second)

	os.RemoveAll(v.FileName() + ".ldb")
	os.RemoveAll(v.FileName() + ".bdb")

	glog.V(3).Infof("Loading volume %d commit file...", v.Id)
	if e = v.load(true, false, v.needleMapKind, 0); e != nil {
		return e
	}
	return nil
}


```

### VacuumVolumeCleanup

``` go

func (vs *VolumeServer) VacuumVolumeCleanup(ctx context.Context, req *volume_server_pb.VacuumVolumeCleanupRequest) (*volume_server_pb.VacuumVolumeCleanupResponse, error) {

	resp := &volume_server_pb.VacuumVolumeCleanupResponse{}

	err := vs.store.CommitCleanupVolume(storage.VolumeId(req.VolumeId))

	if err != nil {
		glog.Errorf("cleanup volume %d: %v", req.VolumeId, err)
	} else {
		glog.V(1).Infof("cleanup volume %d", req.VolumeId)
	}

	return resp, err

}
```



